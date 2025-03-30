package EHentai

import "testing"

func TestDomainFronting(t *testing.T) {
	domainFrontingInterceptor.Enabled = true

	ips := len(domainFrontingInterceptor.IpProvider.(*EhRoundRobinIpProvider).host2Ips["e-hentai.org"])
	t.Logf("total ips: %d", ips)
	for i := range ips {
		_, _, err := fetchPageImageUrl(t.Context(), TEST_PAGE_URL_0)
		if err != nil {
			t.Fatal(err)
		} else {
			t.Logf("fetched %d", i+1)
		}
	}
}
