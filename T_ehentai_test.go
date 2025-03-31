package EHentai

import (
	"context"
	"testing"
	"time"
)

const (
	TEST_FSEARCH_KEYWORD = "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜"
	TEST_GALLERY_GID     = 3138775
	TEST_GALLERY_GTOKEN  = "30b0285f9b"
	TEST_GALLERY_URL     = "https://e-hentai.org/g/3138775/30b0285f9b/"
	TEST_GALLERY_PAGE_0  = 5
	TEST_GALLERY_PAGE_1  = 6
	TEST_PAGE_URL_0      = "https://e-hentai.org/s/859299c9ef/3138775-7"
	TEST_PAGE_URL_1      = "https://e-hentai.org/s/0b2127ea05/3138775-8"
)

var (
	searchTotal   int
	searchResults []FSearchResult
)

var (
	searchDetailTotal   int
	searchDetailResults []GalleryMetadata
)

func getSearch(ctx context.Context) (total int, results []FSearchResult, err error) {
	if searchTotal != 0 && len(searchResults) != 0 {
		return searchTotal, searchResults, nil
	}
	total, results, err = EHSearch(ctx, TEST_FSEARCH_KEYWORD)
	if err != nil {
		return
	}

	searchTotal = total
	searchResults = results
	return
}

func getSearchDetail(ctx context.Context) (total int, results []GalleryMetadata, err error) {
	if searchDetailTotal != 0 && len(searchDetailResults) != 0 {
		return searchDetailTotal, searchDetailResults, nil
	}
	total, results, err = EHSearchDetail(ctx, TEST_FSEARCH_KEYWORD)
	if err != nil {
		return
	}

	searchDetailTotal = total
	searchDetailResults = results
	return
}

func getCoverProviders(ctx context.Context) (providers []coverProvider, err error) {
	_, results, err := getSearch(ctx)
	if err != nil {
		return
	}

	for _, r := range results {
		providers = append(providers, &r)
	}
	return
}

func TestPostGalleryMetadata(t *testing.T) {
	resp, err := PostGalleryMetadata(t.Context(), GIdList{3138775, "30b0285f9b"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
}

func TestPostGalleryToken(t *testing.T) {
	resp, err := PostGalleryToken(t.Context(), PageList{"0b2127ea05", 3138775, 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
}

func TestEHSearch(t *testing.T) {
	total, results, err := getSearch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 {
		t.Fatal("total == 0")
	}

	for _, r := range results {
		t.Log(r.Title)
	}
}

func TestEHSearchDetail(t *testing.T) {
	total, results, err := getSearchDetail(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 {
		t.Fatal("total == 0")
	}
	if len(results) == 0 {
		t.Fatal("empty results")
	}

	for _, r := range results {
		t.Log(r.TitleJpn)
	}
}

func TestDownloadCovers(t *testing.T) {
	providers, err := getCoverProviders(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) == 0 {
		t.Fatal("empty providers")
	}

	for img, err := range DownloadCoversIter(t.Context(), providers...) {
		if err != nil {
			t.Fatal(err)
		}
		if len(img.Data) == 0 {
			t.Fatal("empty data")
		}
		t.Log(img.String())
	}
}

func TestDownloadGallery(t *testing.T) {
	n := 0
	for page, err := range DownloadGalleryIter(t.Context(), TEST_GALLERY_URL, TEST_GALLERY_PAGE_0, TEST_GALLERY_PAGE_1) {
		n++
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Data) == 0 {
			t.Fatal("empty data")
		}
		t.Log(page.String())
	}
	if n != 2 {
		t.Fatal("n != 2")
	}
}

func TestDownloadPages(t *testing.T) {
	n := 0
	for page, err := range DownloadPagesIter(t.Context(), TEST_PAGE_URL_0, TEST_PAGE_URL_1) {
		n++
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Data) == 0 {
			t.Fatal("empty data")
		}
		t.Log(page.String())
	}
	if n != 2 {
		t.Fatal("n != 2")
	}
}

func TestFetchGalleryPageUrls(t *testing.T) {
	pageUrls, err := FetchGalleryPageUrls(t.Context(), TEST_GALLERY_URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(pageUrls) == 0 {
		t.Fatal("empty page urls")
	}

	for _, pageUrl := range pageUrls {
		t.Log(pageUrl)
	}
}

func TestMetaCache(t *testing.T) {
	gMetaCache = newRamCache[int, metaCache](cacheTimeout)
	metadataCacheEnabled = true
	var err error
	var resp *GalleryMetadataResponse
	var pageUrls []string
	resp, err = PostGalleryMetadata(t.Context(), GIdList{TEST_GALLERY_GID, TEST_GALLERY_GTOKEN})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GMetadata) == 0 {
		t.Fatal("len(resp.GMetadata) == 0")
	}
	pageUrls, err = fetchGalleryPages(t.Context(), TEST_GALLERY_URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(pageUrls) == 0 {
		t.Fatal("len(pageUrls) == 0")
	}

	mc := MetaCacheRead(TEST_GALLERY_GID)
	if mc == nil {
		t.Fatal("nil meta cache")
	}
	if mc.gallery == nil {
		t.Fatal("nil meta cache.gallery")
	}
	if len(mc.pageUrls) == 0 {
		t.Fatal("empty meta cache.pageUrls")
	}
	if mc.gallery.GId != TEST_GALLERY_GID {
		t.Fatalf("meta cache.gallery.GId != %d\n", TEST_GALLERY_GID)
	}
}

func TestGalleryCache(t *testing.T) {
	defer func() {
		_ = DeleteCache(TEST_GALLERY_GID)
	}()

	SetDomainFronting(true)
	autoCacheEnabled = true
	var cache *cacheGallery

	// download [TEST_GALLERY_URL]
	// : [TEST_GALLERY_PAGE_0], [TEST_GALLERY_PAGE_1]
	// (write cache)
	for page, err := range DownloadGalleryIter(t.Context(), TEST_GALLERY_URL, TEST_GALLERY_PAGE_0, TEST_GALLERY_PAGE_1) {
		if err != nil {
			t.Fatal(err)
		}
		t.Log(page.String())
	}

	// 缓存的写入是异步的, 等待元数据更新
	<-time.After(time.Second)
	cache = GetCache(TEST_GALLERY_GID)
	if cache == nil {
		t.Fatal("nil cache")
	}
	// cache pages == 2
	if cache.meta.Files.Count != 2 {
		t.Fatal("cache.meta.Files.Count != 2")
	}
	for i, page := range cache.meta.Files.Pages {
		t.Logf("cache.meta.PageUrls[%d]= %v", i, page)
	}

	// download [TEST_PAGE_URL_0], [TEST_PAGE_URL_1]
	// (write cache)
	for page, err := range DownloadPagesIter(t.Context(), TEST_PAGE_URL_0, TEST_PAGE_URL_1) {
		if err != nil {
			t.Fatal(err)
		}
		t.Log(page.String())
	}

	<-time.After(time.Second)
	cache = GetCache(TEST_GALLERY_GID)
	if cache == nil {
		t.Fatal("nil cache")
	}
	// cache pages == 4
	if cache.meta.Files.Count != 4 {
		t.Fatal("cache.meta.Files.Count != 4")
	}
	for i, page := range cache.meta.Files.Pages {
		t.Logf("cache.meta.PageUrls[%d]= %v", i, page)
	}
}
