package EHentai

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"strings"
)

var (
	threads    = 4 // 下载并发数
	retryDepth = 2 // 使用页备链重试次数

	metadataCacheEnabled = true  // 是否启用元数据缓存
	autoCacheEnabled     = false // 是否启用画廊自动缓存
	cacheDir             = DEFAULT_CACHE_DIR
)

// SetCookie 设置 cookie, 访问 exhentai 时必需
func SetCookie(memberId, passHash, igneous, sk string) {
	cookie.set(memberId, passHash, igneous, sk)
}

// SetCookieFromString 从字符串解析 cookie
//
// 返回值 n 为解析的 cookie 数量
func SetCookieFromString(s string) (n int, err error) {
	return cookie.fromString(s)
}

// RegisterIgneousUpdate 注册 igneous 更新时的回调通知
//
// 回调函数将在独立协程中执行
func RegisterIgneousUpdate(f func(igneous string)) {
	igneousUpdateNotifier = f
}

// SetAcceptIgneousMystery 设置是否接受 exhentai 下发的 igneous=mystery
//
// 默认为 false
func SetAcceptIgneousMystery(b bool) {
	acceptMystery = b
}

// SetDomainFronting 设置是否使用域名前置
func SetDomainFronting(b bool) {
	domainFrontingInterceptor.Enabled = b
}

// SetCustomIpProvider 自定义域名前置所使用的 ip 获取器
func SetCustomIpProvider(ipProvider IpProvider) {
	domainFrontingInterceptor.IpProvider = ipProvider
}

func AddInterceptors(interceptors ...Interceptor) {
	interceptorRoundTrip.Interceptors = append(interceptorRoundTrip.Interceptors, interceptors...)
}

func SetInterceptors(interceptors ...Interceptor) {
	interceptorRoundTrip.Interceptors = append(defaultInterceptors, interceptors...)
}

// SetThreads 设置下载并发数
func SetThreads(n int) {
	threads = max(1, n)
}

// SetUseEnvProxy 设置是否使用系统环境变量中的代理
//
// 默认为 true
func SetUseEnvProxy(b bool) {
	if b {
		defaultRoundTripper.Proxy = http.ProxyFromEnvironment
	} else {
		defaultRoundTripper.Proxy = nil
	}
}

// SetRetryDepth 设置重试次数
//
// 默认为 2
//
// 直链下载失败时使用页备链重试
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

// SetAutoCacheEnabled 设置自动缓存启用状态
//
// 默认为 false
//
// 启用时会同时启用元数据缓存
//
// 下载画廊时: 自动缓存所有下载的页
//
// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
func SetAutoCacheEnabled(b bool) {
	autoCacheEnabled = b
	if b {
		metadataCacheEnabled = true
	}
}

// SetCacheDir 设置缓存文件夹路径
//
// 留空默认为 "./EHentaiCache/"
//
// 路径形如 "EHentaiCache/3138775/metadata",
// "EHentaiCache/3138775/1.webp",
// "EHentaiCache/3138775/2.webp"...
func SetCacheDir(dir string) {
	if dir == "" {
		dir = DEFAULT_CACHE_DIR
	}
	cacheDir = dir
}

func EhTagDbOk() bool {
	return ehTagDatabase.Ok()
}

// InitEhTagDb 初始化 EhTagTranslation 数据库
func InitEhTagDb() error {
	return ehTagDatabase.Init()
}

// FreeEhTagDb 释放 EhTagTranslation 数据库
func FreeEhTagDb() {
	ehTagDatabase.Free()
}

// UnmarshalEhTagDb 使用外部的 json 数据反序列化至 EhTagTranslation 数据库
//
// 主要用于 cli 在外部缓存文件
func UnmarshalEhTagDb(data string) error {
	return ehTagDatabase.Unmarshal(data)
}

var ErrInvalidTag = errors.New("invalid tag")

func ParseTags(tags []string) (Tags, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	t := make([]Tag, len(tags))
	for i, tag := range tags {
		tag, err := ParseTag(tag)
		if err != nil {
			return nil, err
		}
		t[i] = tag
	}
	return t, nil
}

func ParseTag(tag string) (Tag, error) {
	s := strings.Split(tag, ":")
	if len(s) != 2 {
		return Tag{}, wrapErr(ErrInvalidTag, tag)
	}
	return Tag{
		Namespace: s[0],
		Name:      s[1],
	}, nil
}

func TranslateTags(tags Tags) Tags {
	return ehTagDatabase.TranslateTags(tags)
}

func TranslateTag(tag Tag) Tag {
	return ehTagDatabase.TranslateTag(tag)
}

// TranslateMulti 翻译多个 tag,
// 输入格式应为: namespace:tag,
// 若数据库未初始化, 则返回入参
func TranslateMulti(tags []string) []string {
	return ehTagDatabase.TranslateMulti(tags)
}

// Translate 翻译 tag,
// 输入格式应为: namespace:tag,
// 若数据库未初始化, 则返回入参
func Translate(tag string) string {
	return ehTagDatabase.Translate(tag)
}

// FSearch 搜索并返回基本信息, results 只有第一页的结果
func FSearch(ctx context.Context, url Url, keyword string, categories ...Category) (total int, results FSearchResults, err error) {
	return queryFSearch(ctx, url, keyword, categories...)
}

// SearchDetail 搜索并返回详细信息, galleries 只有第一页的结果
func SearchDetail(ctx context.Context, url Url, keyword string, categories ...Category) (total int, galleries GalleryMetadatas, err error) {
	return searchDetail(ctx, url, keyword, categories...)
}

// DownloadCoversIter 以迭代器模式通过搜索结果下载封面
func DownloadCoversIter(ctx context.Context, results coverProviders) iter.Seq2[Image, error] {
	return newDownloader(ctx, newImageDownload(results.GetCover())).downloadIterImage()
}

// DownlaodGalleryIter 以迭代器模式下载画廊下所有图片, 下载失败时自动尝试备链
//
// 不传入 pageNums 参数时下载所有页, 传入时按其顺序下载指定页, 重复、越界页将被忽略
func DownloadGalleryIter(ctx context.Context, galleryUrl string, pageNums ...int) iter.Seq2[PageData, error] {
	pageUrls, availableCache, err := initDownloadGallery(ctx, galleryUrl, pageNums...)
	if err != nil {
		return func(yield func(PageData, error) bool) {
			yield(PageData{}, err)
		}
	}
	return newDownloader(ctx, newPageDownload(pageUrls, availableCache)).downloadIterPage()
}

// DownloadPagesIter 以迭代器模式下载画廊某页的图片, 下载失败时自动尝试备链
func DownloadPagesIter(ctx context.Context, pageUrls ...string) iter.Seq2[PageData, error] {
	availableCache, err := initDownloadPages(pageUrls)
	if err != nil {
		return func(yield func(PageData, error) bool) {
			yield(PageData{}, err)
		}
	}
	return newDownloader(ctx, newPageDownload(pageUrls, availableCache)).downloadIterPage()
}

// DownloadCoversTo 完全异步地通过搜索结果下载封面
func DownloadCoversTo(ctx context.Context, results coverProviders, f func(int, Image, error)) error {
	return newDownloader(ctx, newImageDownload(results.GetCover())).downloadImagesTo(f)
}

// DownloadGalleryTo 完全异步地下载画廊下所有图片, 下载失败时自动尝试备链
func DownloadPagesTo(ctx context.Context, pageUrls []string, f func(int, PageData, error)) error {
	availableCache, err := initDownloadPages(pageUrls)
	if err != nil {
		return err
	}
	return newDownloader(ctx, newPageDownload(pageUrls, availableCache)).downloadPagesTo(f)
}

// FetchGalleryDetails 获取画廊详细信息与所有页链接
func FetchGalleryDetails(ctx context.Context, galleryUrl string) (gallery GalleryDetails, err error) {
	return fetchGalleryDetails(ctx, galleryUrl)
}

// EHSearch 搜索 EHentai, results 只有第一页结果
//
// Deprecated: simply use [FSearch] instead
//
//	EHentai.FSearch(ctx, EHentai.EHENTAI_URL, keyword, categories...)
func EHSearch(ctx context.Context, keyword string, categories ...Category) (total int, results FSearchResults, err error) {
	return queryFSearch(ctx, EHENTAI_URL, keyword, categories...)
}

// ExHSearch 搜索 ExHentai, results 只有第一页结果
//
// Deprecated: simply use [FSearch] instead
//
//	EHentai.FSearch(ctx, EHentai.EXHENTAI_URL, keyword, categories...)
func ExHSearch(ctx context.Context, keyword string, categories ...Category) (total int, results FSearchResults, err error) {
	return queryFSearch(ctx, EXHENTAI_URL, keyword, categories...)
}

// EHSearchDetail 搜索 EHentai 并返回详细信息, galleries 只有第一页结果
//
// Deprecated: simply use [SearchDetail] instead
//
//	EHentai.SearchDetail(ctx, EHentai.EHENTAI_URL, keyword, categories...)
func EHSearchDetail(ctx context.Context, keyword string, categories ...Category) (total int, galleries GalleryMetadatas, err error) {
	return searchDetail(ctx, EHENTAI_URL, keyword, categories...)
}

// ExHSearchDetail 搜索 ExHentai 并返回详细信息, galleries 只有第一页结果
//
// Deprecated: simply use [SearchDetail] instead
//
//	EHentai.SearchDetail(ctx, EHentai.EXHENTAI_URL, keyword, categories...)
func ExHSearchDetail(ctx context.Context, keyword string, categories ...Category) (total int, galleries GalleryMetadatas, err error) {
	return searchDetail(ctx, EXHENTAI_URL, keyword, categories...)
}

// FetchGalleryPageUrls 获取画廊下所有页链接
//
// Deprecated: use [FetchGalleryDetails] [GalleryDetails.PageUrls] instead
func FetchGalleryPageUrls(ctx context.Context, galleryUrl string) (pageUrls []string, err error) {
	return fetchGalleryPages(ctx, galleryUrl)
}
