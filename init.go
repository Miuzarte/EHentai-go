package EHentai

import (
	"embed"
	"encoding/json"
	"os"
)

const (
	ENV_COOKIE = "EHENTAI_COOKIE"
	// OR
	ENV_COOKIE_IPB_MEMBER_ID = "EHENTAI_COOKIE_IPB_MEMBER_ID"
	ENV_COOKIE_IPB_PASS_HASH = "EHENTAI_COOKIE_IPB_PASS_HASH"
	ENV_COOKIE_IGNEOUS       = "EHENTAI_COOKIE_IGNEOUS"
	ENV_COOKIE_SK            = "EHENTAI_COOKIE_SK"
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
	domainFrontingInterceptor.IpProvider.(*EhRoundRobinIpProvider).h2IpsCopyFrom(host2Ips)
}

// 从环境变量读取 cookie
func init() {
	ehCookie, ok := os.LookupEnv(ENV_COOKIE)
	if ok {
		Cookie.fromString(ehCookie)
	} else {
		Cookie.ipbMemberId = os.Getenv(ENV_COOKIE_IPB_MEMBER_ID)
		Cookie.ipbPassHash = os.Getenv(ENV_COOKIE_IPB_PASS_HASH)
		Cookie.igneous = os.Getenv(ENV_COOKIE_IGNEOUS)
		Cookie.sk = os.Getenv(ENV_COOKIE_SK)
	}
}
