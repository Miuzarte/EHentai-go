package EHentai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"
	"sync"

	"github.com/Miuzarte/EHentai-go/internal/utils"
)

var ErrDownloadUnreachableCase = fmt.Errorf("download: unreachable case")

var (
	ErrNoPageUrlProvided   = errors.New("no page url provided")
	ErrNoImageUrlProvided  = errors.New("no image url provided")
	ErrTooManyGIds         = errors.New("too many gallery ids")
	ErrInvalidContentType  = errors.New("invalid content type")
	ErrEmptyBody           = errors.New("empty body")
	ErrFoundEmptyImageData = errors.New("found empty image data")
)

// 之后不会扩展, 不使用接口
type download struct {
	url   string
	cache *cacheGallery
	page  *PageData
	img   *Image
	err   chan error
}

func (d *download) start(ctx context.Context) {
	var err error
	defer func() { d.err <- err }()

	switch {
	case d.page != nil:
		var page PageData
		if autoCacheEnabled && d.cache != nil {
			// 读缓存
			page, err = d.cache.ReadOne(d.page.PageNum)
			if err == nil {
				d.page = &page
				return
			}
		}

		page, err = downloadPage(ctx, d.url)
		if err == nil {
			d.page = &page
			// 写缓存
			if autoCacheEnabled && d.cache != nil {
				writingWg.Go(func() {
					d.cache.Write(page)
				})
			}
		}
		return

	case d.img != nil:
		var img Image
		img, err = downloadImage(ctx, d.url)
		if err == nil {
			*d.img = img
		}
		return

	default:
		panic(ErrDownloadUnreachableCase)
	}
}

func newPageDownload(urls []string, aCaches map[int]*cacheGallery) (dls []*download) {
	// 获取可写的画廊缓存
	dls = make([]*download, len(urls))
	for i := range urls {
		pu := UrlToPage(urls[i])
		var cache *cacheGallery
		if aCaches != nil {
			cache = aCaches[pu.GalleryId]
		}
		dls[i] = &download{
			url:   urls[i],
			cache: cache,
			page: &PageData{
				Page: pu,
			},
			err: make(chan error, 1),
		}
	}
	return dls
}

func newImageDownload(urls []string) (dls []*download) {
	dls = make([]*download, len(urls))
	for i := range urls {
		dls[i] = &download{
			url: urls[i],
			img: &Image{},
			err: make(chan error, 1),
		}
	}
	return dls
}

type downloader struct {
	ctx    context.Context
	cancel context.CancelFunc
	items  []*download
}

func newDownloader(ctx context.Context, dls []*download) *downloader {
	ctx, cancel := context.WithCancel(ctx)
	return &downloader{
		ctx:    ctx,
		cancel: cancel,
		items:  dls,
	}
}

func (dl *downloader) startBackground() {
	go func() {
		limiter := newLimiter()
		defer limiter.close()

		for _, item := range dl.items {
			select {
			case <-dl.ctx.Done():
				return
			case limiter.acquire() <- struct{}{}:
			}

			go func() {
				defer limiter.release()
				item.start(dl.ctx)
			}()
		}
	}()
}

func (dl *downloader) downloadIterPage() iter.Seq2[PageData, error] {
	dl.startBackground()
	return func(yield func(PageData, error) bool) {
		defer dl.cancel()
		for _, item := range dl.items {
			if !yield(*item.page, <-item.err) {
				return
			}
		}
	}
}

func (dl *downloader) downloadIterImage() iter.Seq2[Image, error] {
	dl.startBackground()
	return func(yield func(Image, error) bool) {
		defer dl.cancel()
		for _, item := range dl.items {
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

// var ErrWrongSliceSize = errors.New("wrong slice size") // ?

func (dl *downloader) downloadPagesTo(f func(int, PageData, error)) error {
	dl.startBackground()
	for i, item := range dl.items {
		go func() {
			err := <-item.err
			if err != nil {
				f(i, PageData{}, err)
				return
			}
			f(i, *item.page, nil)
		}()
	}
	return nil
}

func (dl *downloader) downloadImagesTo(f func(int, Image, error)) error {
	dl.startBackground()
	for i, item := range dl.items {
		go func() {
			err := <-item.err
			if err != nil {
				f(i, Image{}, err)
				return
			}
			f(i, *item.img, nil)
		}()
	}
	return nil
}

func (dl *downloader) downloadPage() ([]PageData, error) {
	results := make([]PageData, 0, len(dl.items))
	for img, err := range dl.downloadIterPage() {
		if err != nil {
			return nil, err
		}
		results = append(results, img)
	}
	return results, nil
}

func (dl *downloader) downloadImage() ([]Image, error) {
	results := make([]Image, 0, len(dl.items))
	for img, err := range dl.downloadIterImage() {
		if err != nil {
			return nil, err
		}
		results = append(results, img)
	}
	return results, nil
}

// initDownloadGallery
// 获取画廊的所有页链接,
// 根据设置尝试从元数据缓存或本地画廊缓存中获取
//
// 同时根据设置 返回可用的/创建新的 本地缓存
//
// 只有一个 gallery, len(cgs) 不会大于 1
func initDownloadGallery(ctx context.Context, galleryUrl string, pageNums ...int) (pageUrls []string, availCache map[int]*cacheGallery, err error) {
	gId := UrlToGallery(galleryUrl).GalleryId

	cg := GetCache(gId)
	dc := DetailsCacheRead(gId)
	if cg != nil {
		pageUrls = pageUrlsToSlice(cg.meta.PageUrls)
		availCache = make(map[int]*cacheGallery, 1)
		availCache[gId] = cg

	} else if dc != nil && len(dc.PageUrls) != 0 {
		pageUrls = dc.PageUrls
	} else {
		pageUrls, err = fetchGalleryPages(ctx, galleryUrl)
		if err != nil {
			return nil, nil, err
		}
	}

	set := make(utils.Set[int])
	pageNums = set.Clean(pageNums)                        // 去重
	pageNums = cleanOutOfRange(len(pageUrls), pageNums)   // 越界检查
	pageUrls = rearange(pageUrls, sliceAdd(pageNums, -1)) // 按页码重排 url

	if autoCacheEnabled && cg == nil {
		// 创建缓存
		cg, err = CreateCacheFromUrl(galleryUrl)
		if err != nil {
			return nil, nil, err
		}
		if cg != nil {
			availCache = make(map[int]*cacheGallery, 1)
			availCache[gId] = cg
		}
	}

	return
}

// initDownloadPages 返回可用的本地缓存
func initDownloadPages(pageUrls []string) (cgs map[int]*cacheGallery, err error) {
	if len(pageUrls) == 0 {
		return nil, wrapErr(ErrNoPageUrlProvided, nil)
	}

	s := collectGIds(pageUrls)

	cgs = make(map[int]*cacheGallery, len(s))
	for gId := range s {
		cg := GetCache(gId)
		if cg != nil {
			cgs[gId] = cg
		}
	}
	return
}

// downloadImage 从图片直链下载
func downloadImage(ctx context.Context, imgUrl string) (img Image, err error) {
	resp, err := httpGet(ctx, imgUrl)
	if err != nil {
		return Image{}, err
	}
	defer resp.Body.Close()

	// image/webp, image/jpeg, image/png
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image") {
		return Image{}, wrapErr(ErrInvalidContentType, fmt.Sprintf("%s, %s", contentType, imgUrl))
	}
	contentTypeSplits := strings.Split(contentType, "/")
	if len(contentTypeSplits) != 2 || len(contentTypeSplits[1]) == 0 {
		return Image{}, wrapErr(ErrInvalidContentType, fmt.Sprintf("%s, %s", contentType, imgUrl))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Image{}, err
	}
	if len(data) == 0 {
		return Image{}, wrapErr(ErrEmptyBody, imgUrl)
	}
	return Image{Data: data, Type: ParseImageType(contentTypeSplits[1]), TypeRaw: contentTypeSplits[1]}, nil
}

// downloadPage 下载画廊某页的图片,
// 下载失败时尝试备链
func downloadPage(ctx context.Context, pageUrl string) (page PageData, err error) {
	page = PageData{Page: UrlToPage(pageUrl)}
	trys := 0
RETRY:
	trys++
	imgUrl, bakPage, err := fetchPageImageUrl(ctx, pageUrl)
	if err != nil {
		return page, err
	}
	img, err := downloadImage(ctx, imgUrl)
	if err != nil {
		if bakPage != "" && err != context.Canceled {
			if trys <= retryDepth {
				pageUrl = bakPage
				goto RETRY
			}
		}
		return page, err
	}
	page.Image = img
	return page, nil
}

type limiter struct {
	sem  chan struct{}
	once sync.Once
}

func newLimiter() *limiter {
	return &limiter{
		sem: make(chan struct{}, threads),
	}
}

func (l *limiter) acquire() chan<- struct{} {
	return l.sem
}

func (l *limiter) release() {
	<-l.sem
}

func (l *limiter) close() {
	l.once.Do(func() {
		close(l.sem)
	})
}
