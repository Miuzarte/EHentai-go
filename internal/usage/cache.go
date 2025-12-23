package usage

import (
	"context"
	"log"

	ehentai "github.com/Miuzarte/EHentai-go"
)

// 设置缓存
func UsageConfigureCache() {
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
}

// 自行管理缓存
func UsageManageCache() {
	// 确保没有启用自动缓存
	ehentai.SetAutoCacheEnabled(false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const gUrl = "https://e-hentai.org/g/3138775/30b0285f9b"
	gallery := ehentai.UrlToGallery(gUrl)

	var gMeta *ehentai.GalleryMetadata
	var pageUrls []string

	resp, err := ehentai.PostGalleryMetadata(ctx, gallery)
	if err != nil {
		log.Fatalln(err)
	}
	gMeta = &resp.GMetadata[0]
	// gMeta 中没有域名信息所以需要单独传
	// gMeta 画廊元数据不可为空
	// pageUrls 为空时会自动获取
	cache, err := ehentai.CreateCache(ehentai.EHENTAI_DOMAIN, gMeta, pageUrls)
	if err != nil {
		log.Fatalln(err)
	}

	// 下载画廊
	for pageData, err := range ehentai.DownloadGalleryIter(ctx, gUrl) {
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(pageData.String())

		// 写入
		// 如果启用了自动缓存, 当前页在下载时就已经被缓存了
		_, err := cache.Write(pageData)
		if err != nil {
			log.Fatalln(err)
		}
	}

	// 也可以直接从 url 创建, 省去手动获取画廊元数据
	cache, err = ehentai.CreateCacheFromUrl(ctx, gUrl)
	if err != nil {
		log.Fatalln(err)
	}

	// 读取已缓存的所有页
	for pageData, err := range cache.ReadIter(nil) {
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(pageData.String())
	}
	// 读取指定页
	// 如果没有缓存 会返回 [EHentai.ErrPageNotCached] 错误
	for pageData, err := range cache.ReadIter([]int{4, 5, 6}) {
		if err != nil {
			if ehentai.UnwrapErr(err).Is(ehentai.ErrPageNotCached) {
				log.Printf("Page %d not cached\n", pageData.PageNum)
				continue
			}
			log.Println(err)
			break
		}
		log.Println(pageData.String())
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
	ehentai.DeleteCache(ehentai.UrlToGallery(gUrl).GalleryId)
}
