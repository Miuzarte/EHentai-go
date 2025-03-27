package EHentai

import (
	"context"
	"iter"
	"time"
)

var (
	cookie     = Cookie{}
	threads    = 4 // 下载并发数
	timeout    = time.Minute * 5
	retryDepth = 2 // 使用页备链重试次数

	metadataCacheEnabled = true  // 是否启用元数据缓存
	autoCacheEnabled     = false // 是否启用画廊自动缓存
	cacheDir             = DEFAULT_CACHE_DIR
)

// SetCookie 设置 cookie, 访问 exhentai 时必需
func SetCookie(memberId, passHash, igneous, sk string) {
	cookie.IpbMemberId = memberId
	cookie.IpbPassHash = passHash
	cookie.Igneous = igneous
	cookie.Sk = sk
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

// SetCacheEnabled 设置自动缓存启用状态
//
// 默认为 false
//
// 下载画廊时: 自动缓存所有下载的页
//
// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
func SetCacheEnabled(b bool) {
	autoCacheEnabled = b
}

// SetCacheDir 设置缓存目录
//
// 路径形如 "3138775/metadata", "3138775/1.webp", "3138775/2.webp"
func SetCacheDir(dir string) {
	if dir == "" {
		dir = DEFAULT_CACHE_DIR
	}
	cacheDir = dir
}

// SetMetadataCacheEnabled 设置元数据缓存启用状态
//
// 默认为 true
//
// 以避免频繁请求官方 api
func SetMetadataCacheEnabled(b bool) {
	metadataCacheEnabled = b
}

// InitEhTagDB 初始化 EhTagTranslation 数据库
func InitEhTagDB() error {
	return database.Init()
}

// EHSearch 搜索 EHentai, results 只有第一页结果
func EHSearch(keyword string, categories ...Category) (total int, results []FSearchResult, err error) {
	return querySearch(EHENTAI_URL, keyword, categories...)
}

// ExHSearch 搜索 ExHentai, results 只有第一页结果
func ExHSearch(keyword string, categories ...Category) (total int, results []FSearchResult, err error) {
	return querySearch(EXHENTAI_URL, keyword, categories...)
}

// EHSearchDetail 搜索 EHentai 并返回详细信息, galleries 只有第一页结果
func EHSearchDetail(keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	return searchDetail(EHENTAI_URL, keyword, categories...)
}

// ExHSearchDetail 搜索 ExHentai 并返回详细信息, galleries 只有第一页结果
func ExHSearchDetail(keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	return searchDetail(EXHENTAI_URL, keyword, categories...)
}

// DownloadCoversIter 以迭代器模式通过搜索结果下载封面
func DownloadCoversIter[T coverProvider](ctx context.Context, results ...T) iter.Seq2[Image, error] {
	urls := make([]string, 0, len(results))
	for _, result := range results {
		urls = append(urls, result.GetCover())
	}
	job := dlJobImage{}
	job.init(urls)
	job.startBackground()
	return job.downloadIter()
}

// DownloadCovers 通过搜索结果下载封面
func DownloadCovers[T coverProvider](ctx context.Context, results ...T) ([]Image, error) {
	imgs := make([]Image, 0, len(results))
	for _, result := range results {
		img, err := downloadImage(ctx, result.GetCover())
		if err != nil {
			return nil, err
		}
		imgs = append(imgs, img)
	}
	return imgs, nil
}

// DownlaodGalleryIter 以迭代器模式下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 pageNums 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownloadGalleryIter(galleryUrl string, pageNums ...int) iter.Seq2[PageData, error] {
	job := newDlJob()
	job.initGalleryUrl(galleryUrl, pageNums...)
	job.startBackground()
	return job.downloadIter()
}

// DownloadPagesIter 以迭代器模式下载画廊某页的图片, 下载失败时自动尝试备链
func DownloadPagesIter(pageUrls ...string) iter.Seq2[PageData, error] {
	job := newDlJob()
	job.initPageUrls(pageUrls)
	job.startBackground()
	return job.downloadIter()
}

// DownloadGallery 下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 pageNums 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownloadGallery(ctx context.Context, galleryUrl string, pageNums ...int) ([]PageData, error) {
	_, pageUrls, err := initDownloadGalleryUrl(galleryUrl, pageNums...)
	if err != nil {
		return nil, err
	}
	return downloadPages(ctx, pageUrls...)
}

// DownloadPages 下载画廊某页的图片, 下载失败时自动尝试备链
func DownloadPages(ctx context.Context, pageUrls ...string) ([]PageData, error) {
	return downloadPages(ctx, pageUrls...)
}

// FetchGalleryPageUrls 获取画廊下所有页链接
func FetchGalleryPageUrls(galleryUrl string) (pageUrls []string, err error) {
	return fetchGalleryPages(galleryUrl)
}
