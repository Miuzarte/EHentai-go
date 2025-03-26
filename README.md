# EHentai-go

EHentai access for go, with EhTagTranslation support, fully leveraging Go's concurrency advantages.

给机器人做着玩的 部分功能有所缺失, 比如搜索没有翻页之类的

## 用法

### 设置 Cookie

```go
// sk 为空时, 搜索结果标题只有英文
EHentai.SetCookie("ipb_member_id", "ipb_pass_hash", "igneous", "sk")
```

### 初始化 [EhTagTranslation](github.com/EhTagTranslation/Database) 数据库

```go
// 在 AMD Ryzen 5600x(6c12t) 上, 解析数据大概耗时 4ms
err := EHentai.InitEhTagDB()
if err != nil {
    panic(err)
}
```

开了就关不掉了, 要更新的话再调用一次

### 设置下载并发数

```go
// 默认为 4, 不建议超过 16
EHentai.SetThreads(4)
```

### 设置超时时间

```go
// 默认为 time.Minute * 5
EHentai.SetTimeout(time.Minute * 5)

// 除了部分带 ctx 参数的导出函数
// , 该超时上下文会用于内部的所有请求

// 手动使用:
ctx, cancel := EHentai.TimeoutCtx()
defer cancel()
```

### 设置 exhentai 域名的 cookie 检查

```go
// 默认为 false
EHentai.SetDomainCheck(false)
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

### 搜索 E(x)Hentai

```go
// 没做翻页, results 可能比 total 要少
total, results, err := EHentai.EHSearch("keyword")
// total, results, err := EHentai.ExHSearch("keyword")
if err != nil {
    panic(err)
}
fmt.Println("Total results:", total)
for _, result := range results {
    fmt.Printf("%+v", result)
}

// 分类搜索:
EHentai.EHSearch("keyword", EHentai.CATEGORY_DOUJINSHI, EHentai.CATEGORY_MANGA)
// 直接合起来应该也行
EHentai.EHSearch("keyword", EHentai.CATEGORY_DOUJINSHI|EHentai.CATEGORY_MANGA)
```

### 搜索 E(x)Hentai, 并通过官方 API 获取画廊的详细信息

```go
total, results, err := EHentai.EHSearchDetail("keyword")
// total, results, err := EHentai.ExHSearchDetail("keyword")
if err != nil {
    panic(err)
}
fmt.Println("Total results:", total)
for _, result := range results {
    fmt.Printf("%+v", result)
}
```

### 以迭代器模式（后台顺序并发）下载画廊下所有图片, 下载失败时会自动尝试 query nl

```go
for data, err := range EHentai.DownloadGallery("https://e-hentai.org/g/3138775/30b0285f9b") {
    if err != nil {
        // 获取画廊信息出错时, 第一次循环就会返回 err 然后跳出循环
        // 如果是下载过程出错, 由外部决定是否继续下载
        fmt.Println(err)
        break
    }
    fmt.Println(len(imgData))
}
```

### 以迭代器模式（后台顺序并发）下载其中一或几页, 下载失败时会自动尝试 query nl

```go
it := EHentai.TestDownloadPagesIter("https://e-hentai.org/s/859299c9ef/3138775-7", "https://e-hentai.org/s/0b2127ea05/3138775-8")
for data, err := range it {
    if err != nil {
        fmt.Println(err)
        break
    }
    fmt.Println(len(imgData))
}
```

### 下载画廊所有图片, 下载失败时会自动尝试 query nl

```go
imgDatas, err := EHentai.DownloadGallery("https://e-hentai.org/g/3138775/30b0285f9b")
if err != nil {
    panic(err)
}
for _, imgData := range imgDatas {
    fmt.Println(len(imgData))
}
```

### 下载其中一或几页, 下载失败时会自动尝试 query nl

```go
imgDatas, err := EHentai.DownloadPages("https://e-hentai.org/s/859299c9ef/3138775-7", "https://e-hentai.org/s/0b2127ea05/3138775-8")
if err != nil {
    panic(err)
}
for _, imgData := range imgDatas {
    fmt.Println(len(imgData))
}
```

### 获取画廊的所有页链接

```go
pageUrls, err := EHentai.FetchGalleryPageUrls("galleryUrl")
if err != nil {
    panic(err)
}
for _, pageUrl := range pageUrls {
    fmt.Println(pageUrl)
}
```

### 获取画廊所有图片直链与备链

```go
imgUrls, bakPages, err := EHentai.FetchGalleryImageUrls("galleryUrl")
if err != nil {
    panic(err)
}
for i := range imgUrls {
    fmt.Println(imgUrls[i])
    fmt.Println(bakPages[i])
}
```

### 获取某页的图片直链与备链

```go
imgUrl, bakPage, err := EHentai.FetchPageImageUrl("pageUrl")
if err != nil {
    panic(err)
}
fmt.Println(imgUrl)
fmt.Println(bakPage)
```

大概逻辑：直链下载失败时, `EHentai.FetchPageImageUrl(bakPage)` 重新获取直链尝试

### 直链下载

```go
imgDatas, err := EHentai.DownloadImages(imgUrls...)
if err != nil {
    panic(err)
}
for _, imgData := range imgDatas {
    fmt.Println(len(imgData))
}
```

不建议使用, `EHentai.DownloadPages()` 会自动使用备链重试
