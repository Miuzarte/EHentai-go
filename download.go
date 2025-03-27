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

type dlPage struct {
	url  string
	page PageData
	err  chan error
}

func (p *dlPage) done() bool {
	return len(p.page.Data) != 0
}

type dlJobPage struct {
	cgs    map[string]*cacheGallery // usableCache
	pages  []*dlPage
	ctx    context.Context
	cancel context.CancelFunc
	err    error
}

func newDlJob() dlJobPage {
	return dlJobPage{
		cgs: make(map[string]*cacheGallery),
	}
}

func (j *dlJobPage) init(pageUrls []string) {
	j.ctx, j.cancel = context.WithCancel(context.Background())

	if j.err != nil {
		return // do nothing
	}

	if len(j.pages) > 0 {
		return
	}

	j.pages = make([]*dlPage, len(pageUrls))
	for i := range pageUrls {
		j.pages[i] = &dlPage{
			url:  pageUrls[i],
			page: PageData{Page: UrlToPage(pageUrls[i])},
			err:  nil, // 开始时再初始化
		}
	}
}

// initGalleryUrl 以画廊 URL 初始化下载任务
func (j *dlJobPage) initGalleryUrl(galleryUrl string, pageNums ...int) {
	var cgs map[string]*cacheGallery
	var pageUrls []string
	cgs, pageUrls, j.err = initDownloadGalleryUrl(galleryUrl, pageNums...)
	if j.err != nil {
		return
	}
	for gId, cg := range cgs {
		j.cgs[gId] = cg
	}

	j.init(pageUrls)
}

// initPageUrls 以页面 URL 初始化下载任务
func (j *dlJobPage) initPageUrls(pageUrls []string) {
	var cgs map[string]*cacheGallery
	cgs, j.err = initDownloadPageUrls(pageUrls...)
	if j.err != nil {
		return
	}
	for gId, cg := range cgs {
		j.cgs[gId] = cg
	}

	j.init(pageUrls)
}

// downloader 在下载前尝试读取缓存
func (j *dlJobPage) downloader(dl *dlPage) error {
	if cg, ok := j.cgs[itoa(dl.page.GalleryId)]; ok {
		if cg.meta.Files.Pages.Exist(dl.page.PageNum) {
			pages, err := cg.Read(dl.page.PageNum)
			if err == nil {
				dl.page = pages[0]
				return nil
			}
		}
	}

	page, err := downloadPage(j.ctx, dl.url)
	if err != nil {
		return err
	}
	dl.page = page
	return nil
}

// startBackground 启动后台下载协程
func (j *dlJobPage) startBackground() {
	if j.err != nil {
		return
	}

	for _, page := range j.pages {
		page.err = make(chan error, 1)
	}

	go func() {
		limiter := newLimiter()
		for _, page := range j.pages {
			if page.done() {
				// 已完成的直接关闭 err
				close(page.err)
				continue
			}

			limiter.acquire()
			go func(dl *dlPage) {
				defer limiter.release()
				dl.err <- j.downloader(dl)
			}(page)
		}
	}()
}

// downloadIter 以迭代器模式下载
func (j *dlJobPage) downloadIter() iter.Seq2[PageData, error] {
	return func(yield func(PageData, error) bool) {
		defer j.cancel()

		if j.err != nil {
			yield(PageData{}, j.err)
			return
		}
		for _, dl := range j.pages {
			err, ok := <-dl.err
			if !ok { // 已下载, 下载方关闭了 err
				continue
			}

			if !yield(dl.page, err) {
				return
			}
		}
	}
}

type dlImage struct {
	url string
	img Image
	err chan error
}

func (i *dlImage) done() bool {
	return len(i.img.Data) != 0
}

type dlJobImage struct {
	images []*dlImage
	ctx    context.Context
	cancel context.CancelFunc
	err    error
}

func (j *dlJobImage) init(imgUrls []string) {
	j.ctx, j.cancel = context.WithCancel(context.Background())

	if j.err != nil {
		return // do nothing
	}

	if len(j.images) > 0 {
		return
	}

	j.images = make([]*dlImage, len(imgUrls))
	for i := range imgUrls {
		j.images[i] = &dlImage{
			url: imgUrls[i],
			err: nil, // 开始时再初始化
		}
	}
}

func (j *dlJobImage) downloader(dl *dlImage) error {
	img, err := downloadImage(j.ctx, dl.url)
	if err != nil {
		return err
	}
	dl.img = img
	return nil
}

func (j *dlJobImage) startBackground() {
	if j.err != nil {
		return
	}

	for _, img := range j.images {
		img.err = make(chan error, 1)
	}

	go func() {
		limiter := newLimiter()
		for _, img := range j.images {
			if img.done() {
				// 已完成的直接关闭 err
				close(img.err)
				continue
			}

			limiter.acquire()
			go func(dl *dlImage) {
				defer limiter.release()
				dl.err <- j.downloader(dl)
			}(img)
		}
	}()
}

func (j *dlJobImage) downloadIter() iter.Seq2[Image, error] {
	return func(yield func(Image, error) bool) {
		defer j.cancel()

		if j.err != nil {
			yield(Image{}, j.err)
			return
		}
		for _, dl := range j.images {
			err, ok := <-dl.err
			if !ok { // 已下载, 下载方关闭了 err
				continue
			}

			if !yield(dl.img, err) {
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
func initDownloadGalleryUrl(galleryUrl string, pageNums ...int) (cgs map[string]*cacheGallery, pageUrls []string, err error) {
	defer func() { pageUrls = partsDownloadHelper(pageUrls, pageNums) }()

	gId := itoa(UrlToGallery(galleryUrl).GalleryId)
	cg := GetCache(gId)
	if cg != nil && len(cg.meta.Pages) != 0 { // 尝试获取缓存中的 pageUrls
		pageUrls = cg.meta.Pages
		cgs = make(map[string]*cacheGallery, 1)
		cgs[gId] = cg
		return
	}

	pageUrls, err = fetchGalleryPages(galleryUrl)
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
	R := retryDepth
retry:
	imgUrl, bakPage, err := fetchPageImageUrl(pageUrl)
	if err != nil {
		return PageData{}, err
	}
	img, err := downloadImage(ctx, imgUrl)
	if err != nil {
		if bakPage != "" {
			pageUrl = bakPage
			R--
			goto retry
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
