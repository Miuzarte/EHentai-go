package usage

import (
	"log"

	"github.com/Miuzarte/EHentai-go"
)

// 设置 Cookie (初始化时会尝试读取环境变量)
func UsageConfigureCookie() {
	// 环境变量:
	// "EHENTAI_COOKIE"
	// OR
	// "EHENTAI_COOKIE_IPB_MEMBER_ID"
	// "EHENTAI_COOKIE_IPB_PASS_HASH"
	// "EHENTAI_COOKIE_IGNEOUS"
	// "EHENTAI_COOKIE_SK"

	// igneous 为空时, 会尝试使用 exh 下发的 igneous
	// sk 为空时, 搜索结果标题只有英文
	EHentai.SetCookie("ipb_member_id", "ipb_pass_hash", "igneous", "sk")
	// 也可以直接设置字符串
	EHentai.SetCookieFromString("ipb_member_id=123; ipb_pass_hash=abc; igneous=456; sk=efg")

	// 注册 igneous 更新回调
	EHentai.RegisterIgneousUpdate(func(igneous string) {
		log.Printf("igneous updated: %s\n", igneous)
	})
	// 允许 igneous 被下发覆盖为 "mystery", 默认 false
	EHentai.SetAcceptIgneousMystery(false)
}
