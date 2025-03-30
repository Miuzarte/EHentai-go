# EHentai-go

EHentai access for go, with EhTagTranslation support, fully leveraging Go's concurrency advantages.

给机器人做着玩的 部分功能有所缺失, 比如搜索没有翻页之类的

## Features

- 完全并发, 可配置并发数
- 集成 [EhTagTranslation](github.com/EhTagTranslation/Database)
- 指定下载画廊的某(几)页
- 本地画廊缓存
- 域名前置

## 用法

### 设置 Cookie (初始化时会尝试读取环境变量)

```go
// "EHENTAI_COOKIE"
// OR
// "EHENTAI_COOKIE_IPB_MEMBER_ID"
// "EHENTAI_COOKIE_IPB_PASS_HASH"
// "EHENTAI_COOKIE_IGNEOUS"
// "EHENTAI_COOKIE_SK"

// sk 为空时, 搜索结果标题只有英文
EHentai.SetCookie("ipb_member_id", "ipb_pass_hash", "igneous", "sk")
// 也可以直接设置字符串
// EHentai.SetCookieFromString("ipb_member_id=123; ipb_pass_hash=abc; igneous=456; sk=efg")
```

### 初始化 [EhTagTranslation](github.com/EhTagTranslation/Database) 数据库

```go
tStart := time.Now()
// 在 AMD Ryzen 5600x(6c12t) 上, 解析数据大概耗时 4ms
// 开了就关不掉了, 要更新的话再调用一次
err := EHentai.InitEhTagDB()
if err != nil {
    panic(err)
}
fmt.Printf("InitEhTagDB took %s\n", time.Since(tStart))
```

### 设置域名前置

```go
// 默认为 false
EHentai.SetDomainFronting(false)

// 自定义域名前置所使用的 ip 获取器
// type IpProvider interface {
//     Supports(host string) bool
//     NextIp(host string) string
//     AddUnavailableIp(host, ip string)
// }
EHentai.SetCustomIpProvider(IpProvider(nil))
```

### 设置下载并发数

```go
// 默认为 4
EHentai.SetThreads(4)
```

### 设置 query nl 的重试次数

```go
// 默认只尝试两次
EHentai.SetRetryDepth(2)
```

以 `s/b7a3ead2d6/3138775-24` 为例, 图片加载失败时, 页面会根据 `#loadfail` 的内容前往新页面 `s/b7a3ead2d6/3138775-24?nl=45453-483314`

```html
<a href="#" id="loadfail" onclick="return nl('45453-483314')">Reload broken image</a>
```

### 设置缓存

```go
// 设置元数据缓存启用状态
// 默认为 true
// 以避免频繁请求官方 api
EHentai.SetMetadataCacheEnabled(true)

// 设置自动缓存启用状态
// 默认为 false
// 启用时会同时启用元数据缓存
// 下载画廊时: 自动缓存所有下载的页
// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
EHentai.SetAutoCacheEnabled(false)

// 设置缓存文件夹路径
// 留空默认为 "./EHentaiCache/"
// 路径形如 "EHentaiCache/3138775/metadata",
// "EHentaiCache/3138775/1.webp",
// "EHentaiCache/3138775/2.webp"...
EHentai.SetCacheDir("path/to/cache")
```

### 搜索 E(x)Hentai

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 没做翻页, results 可能比 total 要少
total, results, err := EHentai.EHSearch(ctx, "keyword")
// total, results, err := EHentai.ExHSearch(ctx, "keyword")
if err != nil {
    panic(err)
}
fmt.Printf("Total results: %d\n", total)
for _, result := range results {
    fmt.Println(result.Title)
}

// 也可以分类搜索
EHentai.EHSearch(ctx, "keyword", EHentai.CATEGORY_DOUJINSHI, EHentai.CATEGORY_MANGA)
// EHentai.EHSearch(ctx, "keyword", EHentai.CATEGORY_DOUJINSHI|EHentai.CATEGORY_MANGA)

// 搜索同时通过官方 api 获取详细信息
EHentai.EHSearchDetail(ctx, "keyword")
// ExHSearchDetail(ctx, "keyword")
```

### 下载画廊 / 下载页

```go
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
for pageData, err := range EHentai.DownloadGalleryIter(ctx, gUrl) {
    if err != nil {
        fmt.Println(err)
        // 获取画廊信息出错时, 第一次循环就会返回 err 然后跳出
        // 如果是下载过程出错, 可以由外部决定是否取消下载
        break
        // continue
    }
    fmt.Println(pageData.String())
}
// 下载画廊中的指定页
for pageData, err := range EHentai.DownloadGalleryIter(ctx, gUrl, 9, 10, 11) {
    _ = pageData
    _ = err
}
// 下载画廊页
for pageData, err := range EHentai.DownloadPagesIter(ctx, pageUrls...) {
    _ = pageData
    _ = err
}

// 下载全部一起返回:
pageDatas, err := EHentai.DownloadGallery(ctx, gUrl)
_ = pageDatas
_ = err
_, _ = EHentai.DownloadGallery(ctx, gUrl, 9, 10, 11)
_, _ = EHentai.DownloadPages(ctx, pageUrls...)
```

### 获取画廊下所有页链接

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

gUrl := "https://e-hentai.org/g/3138775/30b0285f9b"
pageUrls, err := EHentai.FetchGalleryPageUrls(ctx, gUrl)
if err != nil {
    panic(err)
}
for _, pageUrl := range pageUrls {
    fmt.Println(pageUrl)
}
```

### 自行管理缓存

见 [T_usage_test.go:UsageManageCache](T_usage_test.go#L160)
