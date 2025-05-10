package EHentai

import (
	"embed"
	"encoding/json"
	"os"
	"strconv"
)

const (
	ENV_COOKIE = "EHENTAI_COOKIE"
	// OR
	ENV_COOKIE_IPB_MEMBER_ID = "EHENTAI_COOKIE_IPB_MEMBER_ID"
	ENV_COOKIE_IPB_PASS_HASH = "EHENTAI_COOKIE_IPB_PASS_HASH"
	ENV_COOKIE_IGNEOUS       = "EHENTAI_COOKIE_IGNEOUS"
	ENV_COOKIE_SK            = "EHENTAI_COOKIE_SK"

	ENV_DOMAIN_FRONTING = "EHENTAI_DOMAIN_FRONTING"
)

//go:embed embed/*
var eFs embed.FS

// 初始化默认 ip 列表
func init() {
	host2Ips := make(map[string][]string)
	f, err := eFs.Open("embed/eh_host2ips.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&host2Ips)
	if err != nil {
		panic(err)
	}
	domainFrontingInterceptor.IpProvider.(*EHRoundRobinIpProvider).h2IpsCopyFrom(host2Ips)
}

// 读取环境变量
func init() {
	// cookie
	ehCookie, ok := os.LookupEnv(ENV_COOKIE)
	if ok {
		cookie.fromString(ehCookie)
	} else {
		cookie.ipbMemberId = os.Getenv(ENV_COOKIE_IPB_MEMBER_ID)
		cookie.ipbPassHash = os.Getenv(ENV_COOKIE_IPB_PASS_HASH)
		cookie.igneous = os.Getenv(ENV_COOKIE_IGNEOUS)
		cookie.sk = os.Getenv(ENV_COOKIE_SK)
	}

	// 域名前置, 方便测试
	dfEnabled, err := strconv.ParseBool(os.Getenv(ENV_DOMAIN_FRONTING))
	if err != nil {
		domainFrontingInterceptor.Enabled = dfEnabled
	}
}
