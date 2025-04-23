package EHentai

import (
	"context"
	"testing"
	"time"
)

var (
	ipb_member_id = ""
	ipb_pass_hash = ""
	igneous       = ""
	sk            = ""
)

// 设置 Cookie (初始化时会尝试读取环境变量)
func UsageSetCookie(t *testing.T) {
	// "EHENTAI_COOKIE"
	// OR
	// "EHENTAI_COOKIE_IPB_MEMBER_ID"
	// "EHENTAI_COOKIE_IPB_PASS_HASH"
	// "EHENTAI_COOKIE_IGNEOUS"
	// "EHENTAI_COOKIE_SK"

	// sk 为空时, 搜索结果标题只有英文
	SetCookie(ipb_member_id, ipb_pass_hash, igneous, sk)
	// 也可以直接设置字符串
	// SetCookieFromString("ipb_member_id=123; ipb_pass_hash=abc; igneous=456; sk=efg")
}

// 初始化 [EhTagTranslation](github.com/EhTagTranslation/Database) 数据库
func UsageEhTagDB(t *testing.T) {
	tStart := time.Now()
	// 在 AMD Ryzen 5600x(6c12t) 上, 解析数据大概耗时 4ms
	// 要更新的话再调用一次
	err := InitEhTagDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("InitEhTagDB took %s\n", time.Since(tStart))

	// 释放数据库
	FreeEhTagDB()
}

// 设置域名前置
func UsageSetDomainFronting(t *testing.T) {
	// 默认为 false
	SetDomainFronting(false)

	// 自定义域名前置所使用的 ip 获取器
	// type IpProvider interface {
	//     Supports(host string) bool
	//     NextIp(host string) string
	//     AddUnavailableIp(host, ip string)
	// }
	SetCustomIpProvider(IpProvider(nil))
}

// 设置下载并发数
func UsageSetThreads(t *testing.T) {
	// 默认为 4
	SetThreads(4)
}

// 设置是否使用系统环境变量中的代理
func UsageSetUseEnvPorxy(t *testing.T) {
	// 默认为 true
	// 配合域名前置食用
	SetUseEnvPorxy(true)
}

// 设置 query nl 的重试次数
func UsageSetRetryDepth(t *testing.T) {
	// 默认只尝试两次
	SetRetryDepth(2)
}

// 以 `s/b7a3ead2d6/3138775-24` 为例, 图片加载失败时, 页面会根据 `#loadfail` 的内容前往新页面 `s/b7a3ead2d6/3138775-24?nl=45453-483314`
//
// ```html
// <a href="#" id="loadfail" onclick="return nl('45453-483314')">Reload broken image</a>
// ```

// 设置缓存
func UsageSetCache(t *testing.T) {
	// 设置元数据缓存启用状态
	// 默认为 true
	// 以避免频繁请求官方 api
	SetMetadataCacheEnabled(true)

	// 设置自动缓存启用状态
	// 默认为 false
	// 启用时会同时启用元数据缓存
	// 下载画廊时: 自动缓存所有下载的页
	// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
	SetAutoCacheEnabled(false)

	// 设置缓存文件夹路径
	// 留空默认为 "./EHentaiCache/"
	// 路径形如 "EHentaiCache/3138775/metadata",
	// "EHentaiCache/3138775/1.webp",
	// "EHentaiCache/3138775/2.webp"...
	SetCacheDir("path/to/cache")
}

// 搜索 E(x)Hentai
func UsageEHSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 没做翻页, results 可能比 total 要少
	total, results, err := EHSearch(ctx, "keyword")
	// total, results, err := ExHSearch(ctx, "keyword")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Total results: %d\n", total)
	for _, result := range results {
		t.Log(result.Title)
	}

	// 也可以分类搜索
	EHSearch(ctx, "keyword", CATEGORY_DOUJINSHI, CATEGORY_MANGA)
	// EHSearch(ctx, "keyword", CATEGORY_DOUJINSHI|CATEGORY_MANGA)

	// 搜索同时通过官方 api 获取详细信息
	EHSearchDetail(ctx, "keyword")
	// ExHSearchDetail(ctx, "keyword")
}

// 下载画廊 / 下载页
func UsageDownload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gUrl := "https://e-hentai.org/g/3138775/30b0285f9b"
	pageUrls := []string{
		"https://e-hentai.org/s/859299c9ef/3138775-7",
		"https://e-hentai.org/s/0b2127ea05/3138775-8",
	}

	// 两种下载方式都是一样的根据线程数并发下载

	// 以迭代器模式:
	// 下载整个画廊
	for pageData, err := range DownloadGalleryIter(ctx, gUrl) {
		if err != nil {
			t.Log(err)
			// 获取画廊信息出错时, 第一次循环就会返回 err 然后跳出
			// 如果是下载过程出错, 可以由外部决定是否取消下载
			break
			// continue
		}
		t.Log(pageData.String())
	}
	// 下载画廊中的指定页
	for pageData, err := range DownloadGalleryIter(ctx, gUrl, 9, 10, 11) {
		_ = pageData
		_ = err
	}
	// 下载画廊页
	for pageData, err := range DownloadPagesIter(ctx, pageUrls...) {
		_ = pageData
		_ = err
	}

	// 下载全部一起返回:
	pageDatas, err := DownloadGallery(ctx, gUrl)
	_ = pageDatas
	_ = err
	_, _ = DownloadGallery(ctx, gUrl, 9, 10, 11)
	_, _ = DownloadPages(ctx, pageUrls...)
}

// 获取画廊下所有页链接
func UsageFetchGalleryPageUrls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gUrl := "https://e-hentai.org/g/3138775/30b0285f9b"
	pageUrls, err := FetchGalleryPageUrls(ctx, gUrl)
	if err != nil {
		t.Fatal(err)
	}
	for _, pageUrl := range pageUrls {
		t.Log(pageUrl)
	}
}

// 自行管理缓存
func UsageManageCache(t *testing.T) {
	// 确保没有启用自动缓存
	SetAutoCacheEnabled(false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const gUrl = "https://e-hentai.org/g/3138775/30b0285f9b"
	gallery := UrlToGallery(gUrl)

	var gMeta *GalleryMetadata
	var pageUrls []string

	resp, err := PostGalleryMetadata(ctx, gallery)
	if err != nil {
		t.Fatal(err)
	}
	gMeta = &resp.GMetadata[0]
	// gMeta 中没有域名信息所以需要单独传
	// gMeta 画廊元数据不可为空
	// pageUrls 为空时会自动获取
	cache, err := CreateCache(EHENTAI_DOMAIN, gMeta, pageUrls)
	if err != nil {
		t.Fatal(err)
	}

	// 下载画廊
	for pageData, err := range DownloadGalleryIter(ctx, gUrl) {
		if err != nil {
			t.Log(err)
			break
		}
		t.Log(pageData.String())

		// 写入
		// 如果启用了自动缓存, 当前页在下载时就已经被缓存了
		_, err := cache.Write(pageData)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 也可以直接从 url 创建, 省去手动获取画廊元数据
	cache, err = CreateCacheFromUrl(gUrl)
	if err != nil {
		t.Fatal(err)
	}

	// 读取已缓存的所有页
	for pageData, err := range cache.ReadIter(nil) {
		if err != nil {
			t.Log(err)
			break
		}
		t.Log(pageData.String())
	}
	// 读取指定页, 如果没有缓存, 会返回 [ErrPageNotCached] 错误
	for pageData, err := range cache.ReadIter([]int{4, 5, 6}) {
		if err != nil {
			if UnwrapErr(err).Is(ErrPageNotCached) {
				t.Logf("Page %d not cached\n", pageData.PageNum)
				continue
			}
			t.Log(err)
			break
		}
		t.Log(pageData.String())
	}

	// 其他方法:
	cache.Read(nil) // all
	cache.Read([]int{4, 5, 6})
	cache.ReadOne(7)

	pageUrls = []string{
		"https://e-hentai.org/s/859299c9ef/3138775-7",
		"https://e-hentai.org/s/0b2127ea05/3138775-8",
	}
	cache.ReadByPageUrl(pageUrls[0])

	// pageUrls 不可为空
	// 为了方便起见,
	// len(pageDatas) == len(pageUrls) 且顺序一致,
	// 是否命中缓存只需检查 len(pageDatas[i].Image.Data)
	cache.ReadByPageUrls(pageUrls)

	// 删除某些页
	cache.DeletePages(nil) // all
	cache.DeletePages([]int{4, 5, 6})

	// 删除整个画廊缓存, 包括元数据
	DeleteCache(UrlToGallery(gUrl).GalleryId)
}
