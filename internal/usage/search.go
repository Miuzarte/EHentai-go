package usage

import (
	"context"
	"log"

	"github.com/Miuzarte/EHentai-go"
)

// 搜索 E(x)Hentai
func UsageSearch() {
	const keyword = "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 没做翻页, results 可能比 total 要少
	total, results, err := EHentai.FSearch(ctx, EHentai.EHENTAI_URL, keyword)
	// total, results, err := EHentai.FSearch(ctx, EHentai.EXHENTAI_URL, keyword)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Total results: %d\n", total)
	for _, result := range results {
		log.Println(result.Title)
	}

	// 两种传法
	cate1 := EHentai.CATEGORY_DOUJINSHI | EHentai.CATEGORY_MANGA
	cate2 := []EHentai.Category{EHentai.CATEGORY_DOUJINSHI, EHentai.CATEGORY_MANGA}

	// 也可以分类搜索
	EHentai.FSearch(ctx, EHentai.EHENTAI_URL, keyword, cate1)
	EHentai.FSearch(ctx, EHentai.EXHENTAI_URL, keyword, cate2...)

	// 搜索同时通过官方 api 获取详细信息
	EHentai.SearchDetail(ctx, EHentai.EHENTAI_URL, keyword, cate1)
	EHentai.SearchDetail(ctx, EHentai.EXHENTAI_URL, keyword, cate2...)
}
