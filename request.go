package EHentai

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	netUrl "net/url"
	"sync"
	"time"
)

var domainFrontingInterceptor = &DomainFrontingInterceptor{
	Enabled: false,
	IpProvider: &EHRoundRobinIpProvider{
		host2Ips:       make(map[string][]string),
		ipsIndex:       make(map[string]int),
		unavailableIps: make(map[string]time.Time),
	},
}

var defaultRoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: func(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
		return dialer.DialContext
	}(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}),
	TLSClientConfig: &tls.Config{
		// skip verify certificate for domain fronting
		InsecureSkipVerify: true,
	},
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var defaultInterceptors = []Interceptor{domainFrontingInterceptor}

var interceptorRoundTrip = &InterceptorRoundTrip{
	RoundTripper: defaultRoundTripper,
	Interceptors: defaultInterceptors,
}

// igneousUpdateNotifier igneous 更新时回调通知
var (
	igneousUpdateNotifier func(igneous string)
	acceptMystery = false
)

var cookie = &cookieManager{}

var httpClient = &http.Client{
	Transport: interceptorRoundTrip,
	Jar:       cookie,
}

type cookieManager struct {
	ipbMemberId string
	ipbPassHash string
	igneous     string
	sk          string // 不给的话搜索结果只有英文
	cookies     []*http.Cookie
}

func (c *cookieManager) rebuildCookies() {
	if c.ipbMemberId == "" || c.ipbPassHash == "" {
		c.cookies = nil
		return
	}

	c.cookies = make([]*http.Cookie, 0, 4)
	c.cookies = append(c.cookies, &http.Cookie{Name: "ipb_member_id", Value: c.ipbMemberId})
	c.cookies = append(c.cookies, &http.Cookie{Name: "ipb_pass_hash", Value: c.ipbPassHash})
	if c.igneous != "" {
		c.cookies = append(c.cookies, &http.Cookie{Name: "igneous", Value: c.igneous})
	}
	if c.sk != "" {
		c.cookies = append(c.cookies, &http.Cookie{Name: "sk", Value: c.sk})
	}
}

func (c *cookieManager) SetCookies(u *netUrl.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.Name == "igneous" {
			if cookie.Value != "mystery" || acceptMystery {
				c.igneous = cookie.Value
				c.rebuildCookies()
			}
			if igneousUpdateNotifier != nil {
				go igneousUpdateNotifier(cookie.Value)
			}
		}
	}
}

func (c *cookieManager) Cookies(u *netUrl.URL) (cookies []*http.Cookie) {
	domain := extractMainDomain(u.Host)
	if domain != EHENTAI_DOMAIN && domain != EXHENTAI_DOMAIN {
		return nil
	}
	if c.ipbMemberId == "" || c.ipbPassHash == "" {
		return nil
	}
	return c.cookies
}

func (c *cookieManager) set(memberId, passHash, igneous, sk string) {
	c.ipbMemberId = memberId
	c.ipbPassHash = passHash
	c.igneous = igneous
	c.sk = sk
	c.rebuildCookies()
}

func (c *cookieManager) fromString(cookieStr string) (n int, err error) {
	cookies, err := http.ParseCookie(cookieStr)
	if err != nil {
		return 0, err
	}
	defer c.rebuildCookies()
	for _, cookie := range cookies {
		switch cookie.Name {
		case "ipb_member_id":
			c.ipbMemberId = cookie.Value
			n++
		case "ipb_pass_hash":
			c.ipbPassHash = cookie.Value
			n++
		case "igneous":
			c.igneous = cookie.Value
			n++
		case "sk":
			c.sk = cookie.Value
			n++
		}
	}
	return
}

func (c *cookieManager) String() (s string) {
	if c.ipbMemberId == "" || c.ipbPassHash == "" {
		return ""
	}
	s = "ipb_member_id=" + c.ipbMemberId + "; ipb_pass_hash=" + c.ipbPassHash
	if c.igneous != "" {
		s += "; igneous=" + c.igneous
	}
	if c.sk != "" {
		s += "; sk=" + c.sk
	}
	return s
}

// InterceptorRoundTrip implements [http.RoundTripper],
// but it WILL modify the request and interpret the response
type InterceptorRoundTrip struct {
	RoundTripper http.RoundTripper
	Interceptors []Interceptor
}

func (i *InterceptorRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	err := i.OnRequest(req)
	if err != nil {
		return nil, err
	}

	resp, err := i.RoundTripper.RoundTrip(req)
	if err != nil {
		i.OnError(req, err)
		return resp, err
	}

	err = i.OnResponse(resp)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	return resp, nil
}

func (i *InterceptorRoundTrip) OnRequest(req *http.Request) error {
	for _, interceptor := range i.Interceptors {
		if err := interceptor.OnRequest(req); err != nil {
			return err
		}
	}
	return nil
}

func (i *InterceptorRoundTrip) OnResponse(resp *http.Response) error {
	for _, interceptor := range i.Interceptors {
		if err := interceptor.OnResponse(resp); err != nil {
			return err
		}
	}
	return nil
}

func (i *InterceptorRoundTrip) OnError(req *http.Request, err error) {
	for _, interceptor := range i.Interceptors {
		interceptor.OnError(req, err)
	}
}

// Interceptor is used to modify the request and interpret the response
//
// Implemented by [DomainFrontingInterceptor]
type Interceptor interface {
	OnRequest(req *http.Request) error    // 返回 error 可中断请求
	OnResponse(resp *http.Response) error // 返回 error 可中断请求
	OnError(req *http.Request, err error)
}

// for context kv
type (
	DomainFrontingCtxKey   struct{}
	DomainFrontingCtxValue struct {
		Host string `json:"host"`
		Ip   string `json:"ip"`
	}
)

// DomainFrontingInterceptor 用于实现 e-hentai 的域名前置功能
type DomainFrontingInterceptor struct {
	Enabled    bool
	IpProvider IpProvider
}

func (d *DomainFrontingInterceptor) OnRequest(req *http.Request) error {
	if !d.Enabled || req == nil {
		return nil
	}

	host := req.URL.Host
	if !d.IpProvider.Supports(host) {
		return nil
	}

	ip := d.IpProvider.NextIp(host)
	req.URL.Host = ip
	req.Host = host
	req.Header.Set("Host", host)

	*req = *req.WithContext(
		context.WithValue(
			req.Context(),
			DomainFrontingCtxKey{},
			DomainFrontingCtxValue{host, ip},
		),
	)

	return nil
}

func (d *DomainFrontingInterceptor) OnResponse(resp *http.Response) error {
	if !d.Enabled || resp.Request == nil {
		return nil
	}

	value, ok := resp.Request.Context().Value(DomainFrontingCtxKey{}).(DomainFrontingCtxValue)
	if ok && resp.StatusCode >= 400 {
		d.IpProvider.AddUnavailableIp(value.Host, value.Ip)
	}
	return nil
}

func (d *DomainFrontingInterceptor) OnError(req *http.Request, err error) {
	value, ok := req.Context().Value(DomainFrontingCtxKey{}).(DomainFrontingCtxValue)
	if ok {
		d.IpProvider.AddUnavailableIp(value.Host, value.Ip)
	}
}

// IpProvider 用于轮询提供某个域名的 ip 地址
//
// 全都不可用时, NextIp 也应返回下一个 ip
//
// Implemented by [EHRoundRobinIpProvider]
type IpProvider interface {
	Supports(host string) bool
	NextIp(host string) string
	AddUnavailableIp(host, ip string)
}

// EHRoundRobinIpProvider 用于轮询提供 e-hentai 相关域名的 ip
type EHRoundRobinIpProvider struct {
	host2Ips map[string][]string
	ipsIndex map[string]int
	mu1      sync.RWMutex

	unavailableIps map[string]time.Time
	mu2            sync.Mutex
}

// host2IpsCopyFrom 复制 m 的内容到 p.host2Ips, 避免外部修改
func (p *EHRoundRobinIpProvider) host2IpsCopyFrom(m map[string][]string) {
	p.mu1.Lock()
	p.mu2.Lock()
	defer p.mu1.Unlock()
	defer p.mu2.Unlock()

	for k := range p.host2Ips {
		delete(p.host2Ips, k)
	}
	for k := range p.ipsIndex {
		delete(p.ipsIndex, k)
	}
	for k := range p.unavailableIps {
		delete(p.unavailableIps, k)
	}

	for host, ips := range m {
		p.host2Ips[host] = make([]string, 0, len(ips))
		for _, ip := range ips {
			if ip != "" {
				p.host2Ips[host] = append(p.host2Ips[host], ip)
			}
		}
		p.ipsIndex[host] = 0
	}
	p.unavailableIps = make(map[string]time.Time)
}

func (p *EHRoundRobinIpProvider) Supports(host string) bool {
	p.mu1.RLock()
	defer p.mu1.RUnlock()
	ips, ok := p.host2Ips[host]
	return ok && len(ips) > 0
}

func (p *EHRoundRobinIpProvider) NextIp(host string) (ip string) {
	p.mu1.Lock()
	defer p.mu1.Unlock()

	ips := p.host2Ips[host]
	index := p.ipsIndex[host]
	for range ips {
		index = (index + 1) % len(ips)
		unavailableTime, exists := p.unavailableIps[ips[index]]
		if exists && time.Since(unavailableTime).Minutes() < 5 {
			continue
		}
		break
	}

	ip = ips[index]
	p.ipsIndex[host] = (index + 1) % len(ips)
	return ip
}

func (p *EHRoundRobinIpProvider) AddUnavailableIp(host, ip string) {
	p.mu2.Lock()
	defer p.mu2.Unlock()
	_ = host // 这里用不到
	p.unavailableIps[ip] = time.Now()
}
