# EHentai-go

EHentai access for go, with EhTagTranslation support, fully leveraging Go's concurrency advantages.

给机器人做着玩的 部分功能有所缺失, 比如搜索没有翻页之类的

## Features

- 完全并发, 可配置并发数
- 集成 [EhTagTranslation](https://github.com/EhTagTranslation/Database)
- 指定下载画廊的某(几)页
- 本地画廊缓存
- 域名前置

## 用法

### 开始

```bash
go get github.com/Miuzarte/EHentai-go
```

```go
package main
import ehentai "github.com/Miuzarte/EHentai-go"
```

### 设置 Cookie (初始化时会尝试读取环境变量)

```go
// 环境变量:
// "EHENTAI_COOKIE"
// OR
// "EHENTAI_COOKIE_IPB_MEMBER_ID"
// "EHENTAI_COOKIE_IPB_PASS_HASH"
// "EHENTAI_COOKIE_IGNEOUS"
// "EHENTAI_COOKIE_SK"

// igneous 为空时, 会尝试使用 exh 下发的 igneous
// sk 为空时, 搜索结果标题只有英文
ehentai.SetCookie("ipb_member_id", "ipb_pass_hash", "igneous", "sk")
// 也可以直接设置字符串
ehentai.SetCookieFromString("ipb_member_id=123; ipb_pass_hash=abc; igneous=456; sk=efg")

// 注册 igneous 更新回调
ehentai.RegisterIgneousUpdate(func(igneous string) {
    log.Printf("igneous updated: %s\n", igneous)
})
// 允许 igneous 被下发覆盖为 "mystery", 默认 false
ehentai.SetAcceptIgneousMystery(false)
```

### 初始化 [EhTagTranslation](https://github.com/EhTagTranslation/Database) 数据库

```go
tStart := time.Now()
// 在 AMD Ryzen 5600x(6c12t) 上, 解析数据大概耗时 4ms
// 要更新的话再调用一次
err := ehentai.InitEhTagDb()
if err != nil {
    log.Fatalln(err)
}
// 总时间包括从 GitHub 下载数据
log.Printf("InitEhTagDb took %s\n", time.Since(tStart))

// 释放数据库
ehentai.FreeEhTagDb()
```

### 设置下载并发数

```go
// 默认为 4
ehentai.SetThreads(4)
```

### 设置是否使用系统环境变量中的代理

```go
// 默认为 true
// 配合域名前置设置为 false 食用
ehentai.SetUseEnvProxy(true)
```

### 设置 query nl 的重试次数

```go
// 默认只尝试两次
ehentai.SetRetryDepth(2)
```

以 `s/b7a3ead2d6/3138775-24` 为例, 图片加载失败时, 页面会根据 `#loadfail` 的内容前往新页面 `s/b7a3ead2d6/3138775-24?nl=45453-483314`

```html
<a href="#" id="loadfail" onclick="return nl('45453-483314')">Reload broken image</a>
```

### 设置域名前置

抄的[jiangtian616/JHenTai](https://github.com/jiangtian616/JHenTai)

```go
// 默认为 false
ehentai.SetDomainFronting(false)
```

### 自定义域名前置所使用的 ip 获取器

IpProvider 实现示例: [internal/usage/network.go:MyIpProvider](internal/usage/network.go#L56)

```go
// type IpProvider interface {
//     Supports(host string) bool
//     NextIp(host string) string
//     AddUnavailableIp(host, ip string)
// }
myIpProvider := ehentai.IpProvider(&MyIpProvider{})
ehentai.SetCustomIpProvider(myIpProvider)
```

### 设置缓存

```go
// 设置元数据缓存启用状态
// 默认为 true
// 以避免频繁请求官方 api
ehentai.SetMetadataCacheEnabled(true)

// 设置自动缓存启用状态
// 默认为 false
// 启用时会同时启用元数据缓存
// 下载画廊时: 自动缓存所有下载的页
// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
ehentai.SetAutoCacheEnabled(false)

// 设置缓存文件夹路径
// 留空默认为 "./EHentaiCache/"
// 路径形如 "EHentaiCache/3138775/metadata",
// "EHentaiCache/3138775/1.webp",
// "EHentaiCache/3138775/2.webp"...
ehentai.SetCacheDir("path/to/cache")
```

### 自行管理缓存

见 [internal/usage/cache.go:UsageManageCache](internal/usage/cache.go#L33)

### 搜索 E(x)Hentai

```go
const keyword = "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜"

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 没做翻页, results 可能比 total 要少
total, results, err := ehentai.FSearch(ctx, ehentai.EHENTAI_URL, keyword)
// total, results, err := ehentai.FSearch(ctx, ehentai.EXHENTAI_URL, keyword)
if err != nil {
    log.Fatalln(err)
}
log.Printf("Total results: %d\n", total)
for _, result := range results {
    log.Println(result.Title)
}

// 两种传法
cate1 := ehentai.CATEGORY_DOUJINSHI | ehentai.CATEGORY_MANGA
cate2 := []ehentai.Category{ehentai.CATEGORY_DOUJINSHI, ehentai.CATEGORY_MANGA}

// 也可以分类搜索
ehentai.FSearch(ctx, ehentai.EHENTAI_URL, keyword, cate1)
ehentai.FSearch(ctx, ehentai.EXHENTAI_URL, keyword, cate2...)

// 搜索同时通过官方 api 获取详细信息
ehentai.SearchDetail(ctx, ehentai.EHENTAI_URL, keyword, cate1)
ehentai.SearchDetail(ctx, ehentai.EXHENTAI_URL, keyword, cate2...)

// 下载搜索结果封面
for image, err := range ehentai.DownloadCoversIter(ctx, results) {
    if err != nil {
        log.Println(err)
        break
    }
    log.Println(image.String())
}
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
for pageData, err := range ehentai.DownloadGalleryIter(ctx, gUrl) {
    if err != nil {
        log.Println(err)
        // 获取画廊信息出错时, 第一次循环就会返回 err 然后跳出
        // 如果是下载过程出错, 可以由外部决定是否取消下载
        break
        // continue
    }
    log.Println(pageData.String())
}
// 下载画廊中的指定页
for pageData, err := range ehentai.DownloadGalleryIter(ctx, gUrl, 9, 10, 11) {
    _ = pageData
    _ = err
}
// 下载画廊页
for pageData, err := range ehentai.DownloadPagesIter(ctx, pageUrls...) {
    _ = pageData
    _ = err
}

// 下载全部一起返回:
pageDatas, err := ehentai.DownloadGallery(ctx, gUrl)
_ = pageDatas
_ = err
_, _ = ehentai.DownloadGallery(ctx, gUrl, 9, 10, 11)
_, _ = ehentai.DownloadPages(ctx, pageUrls...)
```

### 通过回调函数完全异步地下载

```go
// 以一个GUI程序为例
reader := myReader{}
reader.Ctx, reader.Cancel = context.WithCancel(context.Background())
reader.Images = make([]widgetsImage, reader.Gallery.Length)
go ehentai.DownloadPagesTo(reader.Ctx, reader.Gallery.PageUrls,
    func(i int, pd ehentai.PageData, err error) {
        if err != nil {
            reader.Images[i].Err = err
            log.Printf("page %d error: %v", i, err)
            return
        }
        reader.Images[i].Image, reader.Images[i].Err = pd.Image.Decode()
        reader.Window.Invalidate() // 触发 GUI 重新渲染
    },
)
// ...
```

### 获取画廊详细信息与所有页链接

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

const gUrl = "https://e-hentai.org/g/3138775/30b0285f9b"
galleryDetails, err := ehentai.FetchGalleryDetails(ctx, gUrl)
if err != nil {
    panic(err)
}
log.Println(galleryDetails.Title, galleryDetails.TitleJpn, galleryDetails.Cat)
for _, pageUrl := range galleryDetails.PageUrls {
    log.Println(pageUrl)
}
```
