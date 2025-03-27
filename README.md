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

### 设置 query nl 的重试次数

```go
// 默认只尝试两次
EHentai.SetRetryDepth(2)
```

以 `s/b7a3ead2d6/3138775-24` 为例, 图片加载失败时, 页面会根据 `#loadfail` 的内容前往新页面 `s/b7a3ead2d6/3138775-24?nl=45453-483314`

```html
<a href="#" id="loadfail" onclick="return nl('45453-483314')">Reload broken image</a>
```

### 启用或禁用自动缓存

```go
// 启用或禁用画廊自动缓存 (WIP)
// 下载画廊时: 自动缓存所有下载的页
// 下载页时: 存在该画廊的缓存时, 自动缓存所下载的页
EHentai.SetCacheEnabled(false)

// 设置缓存目录
EHentai.SetCacheDir("path/to/cache")

// 启用或禁用元数据与页链接缓存, 降低调用官方 api 的频率
EHentai.SetMetadataCacheEnabled(true)
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
    fmt.Printf("%+v\n", result)
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
    fmt.Printf("%+v\n", result)
}
```

### 以迭代器模式（后台顺序并发）下载画廊下所有图片, 下载失败时会自动尝试 query nl

```go
for pageData, err := range EHentai.DownloadGallery("https://e-hentai.org/g/3138775/30b0285f9b") {
    if err != nil {
        // 获取画廊信息出错时, 第一次循环就会返回 err 然后跳出循环
        // 如果是下载过程出错, 由外部决定是否继续下载
        fmt.Println(err)
        break
    }
    fmt.Println(len(pageData.Data))
}
```

### 以迭代器模式（后台顺序并发）下载其中一或几页, 下载失败时会自动尝试 query nl

```go
it := EHentai.DownloadPagesIter("https://e-hentai.org/s/859299c9ef/3138775-7", "https://e-hentai.org/s/0b2127ea05/3138775-8")
for pageData, err := range it {
    if err != nil {
        fmt.Println(err)
        break
    }
    fmt.Println(len(pageData.Data))
}
```

### 下载画廊所有图片, 下载失败时会自动尝试 query nl

```go
pages, err := EHentai.DownloadGallery(context.Background(), "https://e-hentai.org/g/3138775/30b0285f9b")
if err != nil {
    panic(err)
}
for _, page := range pages {
    fmt.Println(len(page.Data))
}
```

### 下载其中一或几页, 下载失败时会自动尝试 query nl

```go
pages, err := EHentai.DownloadPages(context.Background(), "https://e-hentai.org/s/859299c9ef/3138775-7", "https://e-hentai.org/s/0b2127ea05/3138775-8")
if err != nil {
    panic(err)
}
for _, page := range pages {
    fmt.Println(len(page.Data))
}
```

### 获取画廊的所有页链接

```go
pageUrls, err := EHentai.FetchGalleryPageUrls("https://e-hentai.org/g/3138775/30b0285f9b")
if err != nil {
    panic(err)
}
for _, pageUrl := range pageUrls {
    fmt.Println(pageUrl)
}
```

### 创建缓存

```go
resp, err := EHentai.PostGalleryMetadata(EHentai.GIdList{3138775, "30b0285f9b"})
if err != nil {
    panic(err)
}

pages, err := EHentai.DownloadGallery(context.Background(), "https://e-hentai.org/g/3138775/30b0285f9b")
if err != nil {
    panic(err)
}

// 默认情况下 pageUrls 已经在缓存里了, 传 nil 即可
cache, err := EHentai.CreateCache(EHentai.EHENTAI_DOMAIN, galleryMetadata, nil)
if err != nil {
    panic(err)
}

n, err := cache.Write(pages...)
if err != nil {
    panic(err)
}

fmt.Printf("%d pages written\n", n)
```

### 读取缓存

```go
pageNums := []int{7, 8}
cache := EHentai.GetCache("3138775")
if cache != nil {
    for _, page := range cache.ReadIter(pageNums...) {
        fmt.Println(len(page.Data))
    }
}
```
