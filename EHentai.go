package EHentai

import (
	"context"
	"iter"
	"time"
	"unsafe"
)

var (
	cookie      = &Cookie{}
	domainCheck = false // 访问 exhentai 域名时的 cookie 检查
	threads     = 4     // 下载并发数
	timeout     = time.Minute * 5
	retryDepth  = 2 // 使用页备链重试次数
)

func SetCookie(memberId, passHash, igneous, sk string) {
	cookie.IpbMemberId = memberId
	cookie.IpbPassHash = passHash
	cookie.Igneous = igneous
	cookie.Sk = sk
}

// 设置访问 exhentai 域名时的 cookie 检查
func SetDomainCheck(b bool) {
	domainCheck = b
}

// SetThreads 设置下载并发数
func SetThreads(n int) {
	threads = n
}

// SetTimeout 设置超时时间
func SetTimeout(d time.Duration) {
	timeout = d
}

// TimeoutCtx 返回带超时的 context
func TimeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// SetRetryDepth 设置重试次数
// , 默认为 2
// , 直链下载失败时使用页备链重试
func SetRetryDepth(depth int) {
	retryDepth = depth
}

// InitEhTagDB 初始化 EhTagTranslation 数据库
func InitEhTagDB() error {
	return database.Init()
}

// EHSearch 搜索 EHentai, results 只有第一页结果
func EHSearch(keyword string, categories ...Category) (total int, results []EhFSearchResult, err error) {
	return querySearch(EHENTAI_URL, keyword, categories...)
}

// ExHSearch 搜索 ExHentai, results 只有第一页结果
func ExHSearch(keyword string, categories ...Category) (total int, results []EhFSearchResult, err error) {
	if !cookie.Ok() {
		return 0, nil, ErrCookieNotSet
	}
	return querySearch(EXHENTAI_URL, keyword, categories...)
}

// EHSearchDetail 搜索 EHentai 并返回详细信息, galleries 只有第一页结果
func EHSearchDetail(keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	total, results, err := EHSearch(keyword, categories...)
	if err != nil {
		return
	}
	list := *(*[]GIdList)(unsafe.Pointer(&results))
	resp, err := PostGalleryMetadata(list...)
	if err != nil {
		return 0, nil, err
	}
	return total, resp.GMetadata, nil
}

// ExHSearchDetail 搜索 ExHentai 并返回详细信息, galleries 只有第一页结果
func ExHSearchDetail(keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	total, results, err := ExHSearch(keyword, categories...)
	if err != nil {
		return
	}
	list := *(*[]GIdList)(unsafe.Pointer(&results))
	resp, err := PostGalleryMetadata(list...)
	if err != nil {
		return 0, nil, err
	}
	return total, resp.GMetadata, nil
}

// PageData carrys page info and data
type PageData struct {
	Page
	Data []byte
}

// DownlaodGalleryIter 以迭代器模式下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 parts 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownlaodGalleryIter(galleryUrl string, parts ...int) iter.Seq2[PageData, error] {
	job := dlJob{}
	err := checkDomain(galleryUrl)
	if err != nil {
		job.err = err
	}
	pageUrls, err := fetchGalleryPages(galleryUrl)
	if err != nil {
		job.err = err
	}

	if len(parts) > 0 {
		parts = removeDuplicates(parts)                      // 去重
		parts = indexesCleanOutOfRange(len(pageUrls), parts) // 越界检查
		pageUrls = sliceRearrange(pageUrls, parts)           // 重排
	}

	job.init(pageUrls)
	job.startBackground()
	return job.downloadIter()
}

// DownloadPagesIter 以迭代器模式下载画廊某页的图片, 下载失败时自动尝试备链
func DownloadPagesIter(pageUrls ...string) iter.Seq2[PageData, error] {
	job := dlJob{}
	err := checkDomain(pageUrls...)
	if err != nil {
		job.err = err
	}

	job.init(pageUrls)
	job.startBackground()
	return job.downloadIter()
}

// DownloadGallery 下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 parts 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownloadGallery(ctx context.Context, galleryUrl string, parts ...int) (imgDatas []PageData, err error) {
	err = checkDomain(galleryUrl)
	if err != nil {
		return nil, err
	}
	pageUrls, err := fetchGalleryPages(galleryUrl)
	if err != nil {
		return nil, err
	}

	if len(parts) > 0 {
		parts = removeDuplicates(parts)                      // 去重
		parts = indexesCleanOutOfRange(len(pageUrls), parts) // 越界检查
		pageUrls = sliceRearrange(pageUrls, parts)           // 重排
	}

	return downloadPages(ctx, pageUrls...)
}

// DownloadPages 下载画廊某页的图片, 下载失败时自动尝试备链
func DownloadPages(ctx context.Context, pageUrls ...string) (imgDatas []PageData, err error) {
	err = checkDomain(pageUrls...)
	if err != nil {
		return nil, err
	}
	return downloadPages(ctx, pageUrls...)
}

// FetchGalleryPageUrls 获取画廊下所有页链接
func FetchGalleryPageUrls(galleryUrl string) (pageUrls []string, err error) {
	err = checkDomain(galleryUrl)
	if err != nil {
		return nil, err
	}
	return fetchGalleryPages(galleryUrl)
}

// FetchGalleryImageUrls 获取画廊下所有图片的直链与页面备链
// , 不建议使用
// , [DownloadPages] 可使用备链自动重试
func FetchGalleryImageUrls(galleryUrl string) (imgUrls []string, bakPages []string, err error) {
	err = checkDomain(galleryUrl)
	if err != nil {
		return nil, nil, err
	}
	pageUrls, err := fetchGalleryPages(galleryUrl)
	if err != nil {
		return nil, nil, err
	}
	for _, pageUrl := range pageUrls {
		imgUrl, bak, err := fetchPageImageUrl(pageUrl)
		if err != nil {
			return nil, nil, err
		}
		imgUrls = append(imgUrls, imgUrl)
		bakPages = append(bakPages, bak)
	}
	return
}

// FetchPageImageUrl 获取画廊某页的直链与页面备链
// , 不建议使用
// , [DownloadPages] 可使用备链自动重试
func FetchPageImageUrl(pageUrl string) (imgUrl string, bakPage string, err error) {
	err = checkDomain(pageUrl)
	if err != nil {
		return "", "", err
	}
	return fetchPageImageUrl(pageUrl)
}

// DownloadImage 使用图片直链下载
// , 不建议使用
// , [DownloadPages] 可使用备链自动重试
func DownloadImages(ctx context.Context, imgUrls ...string) (imgDatas [][]byte, err error) {
	for _, url := range imgUrls {
		var data []byte
		data, err = downloadImage(ctx, url)
		if err != nil {
			return nil, err
		}
		imgDatas = append(imgDatas, data)
	}
	return imgDatas, nil
}
