package EHentai

import (
	"context"
	"testing"
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

func getSearch(ctx context.Context) (total int, results FSearchResults, err error) {
	if searchTotal != 0 && len(searchResults) != 0 {
		return searchTotal, searchResults, nil
	}
	total, results, err = FSearch(ctx, EHENTAI_URL, TEST_FSEARCH_KEYWORD)
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
	total, results, err = SearchDetail(ctx, EHENTAI_URL, TEST_FSEARCH_KEYWORD)
	if err != nil {
		return
	}

	searchDetailTotal = total
	searchDetailResults = results
	return
}

func TestEhApiPostGalleryMetadata(t *testing.T) {
	resp, err := PostGalleryMetadata(t.Context(), GIdList{3138775, "30b0285f9b"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
}

func TestEhApiPostGalleryToken(t *testing.T) {
	resp, err := PostGalleryToken(t.Context(), PageList{"0b2127ea05", 3138775, 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
}

func TestEhSearch(t *testing.T) {
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

func TestEhSearchDetail(t *testing.T) {
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

func TestEhDownloadCovers(t *testing.T) {
	_, results, err := getSearch(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for img, err := range DownloadCoversIter(t.Context(), results) {
		if err != nil {
			t.Fatal(err)
		}
		if len(img.Data) == 0 {
			t.Fatal("empty data")
		}
		t.Log(img.String())
	}
}

func TestEhDownloadGallery(t *testing.T) {
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

func TestEhDownloadPages(t *testing.T) {
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

func TestEhFetchGalleryPageUrls(t *testing.T) {
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
