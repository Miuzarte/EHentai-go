package usage

import (
	"maps"
	"math/rand/v2"
	"slices"
	"sync"

	"github.com/Miuzarte/EHentai-go"
)

// 设置下载并发数
func UsageSetThreads() {
	// 默认为 4
	EHentai.SetThreads(4)
}

// 设置是否使用系统环境变量中的代理
func UsageSetUseEnvPorxy() {
	// 默认为 true
	// 配合域名前置设置为 false 食用
	EHentai.SetUseEnvProxy(true)
}

// 设置 query nl 的重试次数
func UsageSetRetryDepth() {
	// 默认只尝试两次
	EHentai.SetRetryDepth(2)
}

// 以 `s/b7a3ead2d6/3138775-24` 为例, 图片加载失败时, 页面会根据 `#loadfail` 的内容前往新页面 `s/b7a3ead2d6/3138775-24?nl=45453-483314`
//
// ```html
// <a href="#" id="loadfail" onclick="return nl('45453-483314')">Reload broken image</a>
// ```

// 设置域名前置
func UsageSetDomainFronting() {
	// 默认为 false
	EHentai.SetDomainFronting(false)
}

// 自定义域名前置所使用的 ip 获取器
func UsageSetCustomIpProvider() {
	myIpProvider := EHentai.IpProvider(&MyIpProvider{})
	EHentai.SetCustomIpProvider(myIpProvider)
}

//	type IpProvider interface {
//	    Supports(host string) bool
//	    NextIp(host string) string
//	    AddUnavailableIp(host, ip string)
//	}
//
// 简单实现 仅作示例
type MyIpProvider struct {
	host map[string]string
	mu   sync.RWMutex
}

func (mip *MyIpProvider) Supports(host string) bool {
	mip.mu.RLock()
	defer mip.mu.RUnlock()
	_, ok := mip.host[host]
	return ok
}

func (mip *MyIpProvider) NextIp(host string) string {
	mip.mu.RLock()
	defer mip.mu.RUnlock()
	keys := slices.Collect(maps.Keys(mip.host))
	return mip.host[keys[rand.IntN(len(keys))]]
}

func (mip *MyIpProvider) AddUnavailableIp(host, ip string) {
	mip.mu.Lock()
	defer mip.mu.Unlock()
	_ = ip
	delete(mip.host, host)
}
