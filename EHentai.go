package EHentai

import (
	"context"
	"iter"
)

var (
	cookie     = Cookie{}
	threads    = 4 // 下载并发数
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

// SetRetryDepth 设置重试次数
// , 默认为 2
// , 直链下载失败时使用页备链重试
func SetRetryDepth(depth int) {
	retryDepth = depth
}

// SetMetadataCacheEnabled 设置元数据缓存启用状态
//
// 默认为 true
//
// 以避免频繁请求官方 api
func SetMetadataCacheEnabled(b bool) {
	metadataCacheEnabled = b
}

// SetCacheEnabled 设置自动缓存启用状态
//
// 默认为 false
//
// 启用时会同时启用元数据缓存
//
// 下载画廊时: 自动缓存所有下载的页
//
// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
func SetCacheEnabled(b bool) {
	autoCacheEnabled = b
	if b {
		metadataCacheEnabled = true
	}
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

// InitEhTagDB 初始化 EhTagTranslation 数据库
func InitEhTagDB() error {
	return database.Init()
}

// EHSearch 搜索 EHentai, results 只有第一页结果
func EHSearch(ctx context.Context, keyword string, categories ...Category) (total int, results []FSearchResult, err error) {
	return querySearch(ctx, EHENTAI_URL, keyword, categories...)
}

// ExHSearch 搜索 ExHentai, results 只有第一页结果
func ExHSearch(ctx context.Context, keyword string, categories ...Category) (total int, results []FSearchResult, err error) {
	return querySearch(ctx, EXHENTAI_URL, keyword, categories...)
}

// EHSearchDetail 搜索 EHentai 并返回详细信息, galleries 只有第一页结果
func EHSearchDetail(ctx context.Context, keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	return searchDetail(ctx, EHENTAI_URL, keyword, categories...)
}

// ExHSearchDetail 搜索 ExHentai 并返回详细信息, galleries 只有第一页结果
func ExHSearchDetail(ctx context.Context, keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	return searchDetail(ctx, EXHENTAI_URL, keyword, categories...)
}

// DownloadCoversIter 以迭代器模式通过搜索结果下载封面
func DownloadCoversIter(ctx context.Context, results ...coverProvider) iter.Seq2[Image, error] {
	urls := make([]string, 0, len(results))
	for _, result := range results {
		urls = append(urls, result.GetCover())
	}
	job := newDownloader(ctx, newImageDownload(urls))
	job.startBackground()
	return job.downloadIterImage()
}

// DownloadCovers 通过搜索结果下载封面
func DownloadCovers(ctx context.Context, results ...coverProvider) ([]Image, error) {
	urls := make([]string, 0, len(results))
	for _, result := range results {
		urls = append(urls, result.GetCover())
	}
	return downloadImages(ctx, urls...)
}

// DownlaodGalleryIter 以迭代器模式下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 pageNums 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownloadGalleryIter(ctx context.Context, galleryUrl string, pageNums ...int) iter.Seq2[PageData, error] {
	var pageUrls []string
	var err error
	g := UrlToGallery(galleryUrl)
	mCache, ok := MetaCacheRead(g.GalleryId)
	if ok && len(mCache.pageUrls) != 0 {
		pageUrls = mCache.pageUrls
	} else {
		pageUrls, err = fetchGalleryPages(ctx, galleryUrl)
	}
	pageUrls = partsDownloadHelper(pageUrls, pageNums)

	job := newDownloader(ctx, newPageDownload(pageUrls))
	job.firstYieldErr = err
	job.startBackground()
	return job.downloadIterPage()
}

// DownloadPagesIter 以迭代器模式下载画廊某页的图片, 下载失败时自动尝试备链
func DownloadPagesIter(ctx context.Context, pageUrls ...string) iter.Seq2[PageData, error] {
	job := newDownloader(ctx, newPageDownload(pageUrls))
	job.startBackground()
	return job.downloadIterPage()
}

// DownloadGallery 下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 pageNums 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownloadGallery(ctx context.Context, galleryUrl string, pageNums ...int) ([]PageData, error) {
	_, pageUrls, err := initDownloadGalleryUrl(ctx, galleryUrl, pageNums...)
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
func FetchGalleryPageUrls(ctx context.Context, galleryUrl string) (pageUrls []string, err error) {
	return fetchGalleryPages(ctx, galleryUrl)
}
