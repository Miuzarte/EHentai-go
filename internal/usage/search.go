package usage

import (
	"context"
	"log"

	ehentai "github.com/Miuzarte/EHentai-go"
)

// 搜索 E(x)Hentai
func UsageSearch() {
	const keyword = "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 没做翻页, results 可能比 total 要少
	total, results, err := ehentai.FSearch(ctx, ehentai.EHENTAI_URL, keyword)
	// total, results, err := EHentai.FSearch(ctx, EHentai.EXHENTAI_URL, keyword)
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
}
