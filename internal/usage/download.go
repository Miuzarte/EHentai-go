package usage

import (
	"context"
	"image"
	"log"

	ehentai "github.com/Miuzarte/EHentai-go"
)

// 下载画廊 / 下载页
func UsageDownload() {
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
}

// 通过回调函数完全异步地下载
func UsageDownloadAsync() {
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
}

type myReader struct {
	Ctx    context.Context
	Cancel context.CancelFunc

	Gallery ehentai.GalleryDetails
	Cover   widgetsImage
	Images  []widgetsImage

	Window interface{ Invalidate() }
}

type widgetsImage struct {
	Image image.Image // show normally
	Err   error       // != nil: show error on screen
}

// 获取画廊详细信息与所有页链接
func UsageFetchGalleryPageUrls() {
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
}
