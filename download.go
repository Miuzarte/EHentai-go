package EHentai

import (
	"context"
	"fmt"
	"io"
	"iter"
	netUrl "net/url"
	"strings"
	"sync"
)

var ErrDownloadUnreachableCase = fmt.Errorf("download: unreachable case")

// 之后不会扩展, 不使用接口
type download struct {
	url  string
	page *PageData
	img  *Image
	err  chan error
}

func (dl *download) done() bool {
	switch {
	case dl.page != nil:
		return len(dl.page.Data) != 0
	case dl.img != nil:
		return len(dl.img.Data) != 0

	default:
		panic(ErrDownloadUnreachableCase)
	}
}

func (dl *download) start(ctx context.Context) {
	var err error
	defer func() { dl.err <- err }()

	switch {
	case dl.page != nil:
		var page PageData
		page, err = downloadPage(ctx, dl.url)
		if err == nil {
			dl.page = &page
		}
	case dl.img != nil:
		var img Image
		img, err = downloadImage(ctx, dl.url)
		if err == nil {
			dl.img = &img
		}

	default:
		panic(ErrDownloadUnreachableCase)
	}
}

func newPageDownload(urls []string) (dls []*download) {
	dls = make([]*download, len(urls))
	for _, url := range urls {
		dls = append(dls, &download{
			url: url,
			page: &PageData{
				Page: UrlToPage(url),
			},
			err: make(chan error, 1),
		})
	}
	return dls
}

func newImageDownload(urls []string) (dls []*download) {
	dls = make([]*download, len(urls))
	for _, url := range urls {
		dls = append(dls, &download{
			url: url,
			img: &Image{},
			err: make(chan error, 1),
		})
	}
	return dls
}

type downloader struct {
	ctx           context.Context
	cancel        context.CancelFunc
	items         []*download
	firstYieldErr error // 包装外部错误到迭代器中
}

func newDownloader(ctx context.Context, dls []*download) *downloader {
	ctx, cancel := context.WithCancel(ctx)
	return &downloader{
		ctx:    ctx,
		cancel: cancel,
		items:  dls,
	}
}

func (j *downloader) startBackground() {
	if j.firstYieldErr != nil {
		return
	}

	go func() {
		limiter := newLimiter()
		for _, item := range j.items {
			if item.done() {
				close(item.err)
				continue
			}

			limiter.acquire()
			go func() {
				defer limiter.release()
				item.start(j.ctx)
			}()
		}
	}()
}

func (j *downloader) downloadIterPage() iter.Seq2[PageData, error] {
	return func(yield func(PageData, error) bool) {
		defer j.cancel()

		if j.firstYieldErr != nil {
			yield(PageData{}, j.firstYieldErr)
			return
		}

		for _, item := range j.items {
			err, ok := <-item.err
			if !ok {
				continue
			}

			if !yield(*item.page, err) {
				return
			}
		}
	}
}

func (j *downloader) downloadIterImage() iter.Seq2[Image, error] {
	return func(yield func(Image, error) bool) {
		defer j.cancel()

		if j.firstYieldErr != nil {
			yield(Image{}, j.firstYieldErr)
			return
		}

		for _, item := range j.items {
			err, ok := <-item.err
			if !ok {
				continue
			}

			if !yield(*item.img, err) {
				return
			}
		}
	}
}

func partsDownloadHelper(pageUrls []string, pageNums []int) []string {
	if len(pageNums) == 0 || len(pageUrls) == 0 {
		return pageUrls
	}
	pageNums = removeDuplicates(pageNums)               // 去重
	pageNums = cleanOutOfRange(len(pageUrls), pageNums) // 越界检查
	return rearrange(pageUrls, pageNums)                // 按页码重排 url
}

// initDownloadGalleryUrl
//
// 只有一个 gallery, len(cgs) 不会大于 1
func initDownloadGalleryUrl(ctx context.Context, galleryUrl string, pageNums ...int) (cgs map[string]*cacheGallery, pageUrls []string, err error) {
	defer func() { pageUrls = partsDownloadHelper(pageUrls, pageNums) }()

	gId := itoa(UrlToGallery(galleryUrl).GalleryId)
	cg := GetCache(gId)
	if cg != nil && len(cg.meta.Pages) != 0 { // 尝试获取缓存中的 pageUrls
		pageUrls = cg.meta.Pages
		cgs = make(map[string]*cacheGallery, 1)
		cgs[gId] = cg
		return
	}

	pageUrls, err = fetchGalleryPages(ctx, galleryUrl)
	if err != nil {
		return nil, nil, err
	}

	return
}

func initDownloadPageUrls(pageUrls ...string) (cgs map[string]*cacheGallery, err error) {
	if len(pageUrls) == 0 {
		return nil, wrapErr(ErrNoPageProvided, nil)
	}

	s := make(set[string])
	for _, pageUrl := range pageUrls {
		_, _, gId, _ := UrlGetPTokenGIdPNum(pageUrl)
		s[gId] = struct{}{}
	}

	cgs = make(map[string]*cacheGallery, len(s))
	for gId := range s {
		cg := GetCache(gId)
		if cg != nil {
			cgs[gId] = cg
		}
	}
	return
}

type limiter struct {
	sem chan struct{}
	cl  sync.Once
}

func newLimiter() *limiter {
	return &limiter{
		sem: make(chan struct{}, threads),
	}
}

func (l *limiter) acquire() {
	l.sem <- struct{}{}
}

func (l *limiter) release() {
	<-l.sem
}

func (l *limiter) close() {
	l.cl.Do(func() {
		close(l.sem)
	})
}

// downloadImage 从图片直链下载
func downloadImage(ctx context.Context, imgUrl string) (img Image, err error) {
	u, err := netUrl.Parse(imgUrl)
	if err != nil {
		return Image{}, err
	}
	resp, err := httpGet(ctx, u)
	if err != nil {
		return Image{}, err
	}
	defer resp.Body.Close()

	// image/webp, image/jpeg
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image") {
		return Image{}, wrapErr(ErrInvalidContentType, fmt.Sprintf("%s, %s", contentType, imgUrl))
	}
	contentTypeSplits := strings.Split(contentType, "/")
	if len(contentTypeSplits) != 2 {
		return Image{}, wrapErr(ErrInvalidContentType, fmt.Sprintf("%s, %s", contentType, imgUrl))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Image{}, err
	}
	if len(data) == 0 {
		return Image{}, wrapErr(ErrEmptyBody, imgUrl)
	}
	return Image{Data: data, Type: ParseImageType(contentTypeSplits[1])}, nil
}

// downloadImages 并发从图片直链下载
func downloadImages(ctx context.Context, imgUrls ...string) (imgs []Image, err error) {
	if len(imgUrls) == 0 {
		return nil, wrapErr(ErrNoPageUrls, nil)
	}

	imgs = make([]Image, len(imgUrls))
	errs := make(chan error, len(imgUrls))

	wg := sync.WaitGroup{}
	wg.Add(len(imgUrls))

	limiter := newLimiter()
	defer limiter.close()

	for i, url := range imgUrls {
		limiter.acquire()
		go func(i int) {
			defer func() {
				limiter.release()
				wg.Done()
			}()

			data, err := downloadImage(ctx, url)
			imgs[i] = data
			errs <- err
		}(i)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	for i := range imgs {
		if len(imgs[i].Data) == 0 {
			return nil, wrapErr(ErrFoundEmptyImageData, imgUrls[i])
		}
	}

	return imgs, nil
}

// downloadPage 下载画廊某页的图片, 下载失败时尝试备链
func downloadPage(ctx context.Context, pageUrl string) (PageData, error) {
	trys := 0
retry:
	trys++
	imgUrl, bakPage, err := fetchPageImageUrl(ctx, pageUrl)
	if err != nil {
		return PageData{}, err
	}
	img, err := downloadImage(ctx, imgUrl)
	if err != nil {
		if bakPage != "" && err != context.Canceled {
			if trys <= retryDepth {
				pageUrl = bakPage
				goto retry
			}
		}
		return PageData{}, err
	}
	return PageData{Page: UrlToPage(pageUrl), Image: img}, nil
}

// downloadPages 并发下载画廊某页的图片, 下载失败时尝试备链
func downloadPages(ctx context.Context, pageUrls ...string) (imgDatas []PageData, err error) {
	if len(pageUrls) == 0 {
		return nil, wrapErr(ErrNoPageUrls, nil)
	}

	imgDatas = make([]PageData, len(pageUrls))
	errs := make(chan error, len(pageUrls))

	wg := sync.WaitGroup{}
	wg.Add(len(pageUrls))

	limiter := newLimiter()
	defer limiter.close()

	for i, url := range pageUrls {
		limiter.acquire()
		go func(i int) {
			defer func() {
				limiter.release()
				wg.Done()
			}()

			data, err := downloadPage(ctx, url)
			imgDatas[i] = data
			errs <- err
		}(i)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	for i := range imgDatas {
		if len(imgDatas[i].Data) == 0 {
			return nil, wrapErr(ErrFoundEmptyImageData, pageUrls[i])
		}
	}

	return imgDatas, nil
}
