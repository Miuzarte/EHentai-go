package EHentai

import (
	"os"
	"testing"
	"time"
)

const (
	TEST_GALLERY_URL = "https://e-hentai.org/g/3138775/30b0285f9b/"
	TEST_PAGE_URL_0  = "https://e-hentai.org/s/859299c9ef/3138775-7"
	TEST_PAGE_URL_1  = "https://e-hentai.org/s/0b2127ea05/3138775-8"
)

func TestEhQueryFSearch(t *testing.T) {
	// results, err := queryFSearch(EHENTAI_URL, "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("%+v", results)

	SetCookie("", "", "", "")
	_, results, err := querySearch(EXHENTAI_URL, "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", results)
}

func TestFetchGalleryPageUrls(t *testing.T) {
	t.Log(FetchGalleryPageUrls(TEST_GALLERY_URL))
}

func TestFetchGalleryImageUrls(t *testing.T) {
	_, pageUrls, err := initDownloadGalleryUrl(TEST_GALLERY_URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, pageUrl := range pageUrls {
		imgUrl, bak, err := fetchPageImageUrl(pageUrl)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(imgUrl)
		t.Log(bak)
		println()
	}
}

func TestFetchPageImageUrl(t *testing.T) {
	t.Log(fetchPageImageUrl(TEST_PAGE_URL_0))
	t.Log(fetchPageImageUrl(TEST_PAGE_URL_1))
}

func TestDownloadPages(t *testing.T) {
	ctx, cancel := TimeoutCtx()
	defer cancel()
	img, err := DownloadPages(ctx, TEST_PAGE_URL_0, TEST_PAGE_URL_1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(len(img[0].Data))
}

func TestDownlaodGalleryIter(t *testing.T) {
	stop := false
	go func() {
		<-time.After(time.Second * 15)
		stop = true
	}()
	for page, err := range DownloadGalleryIter(TEST_GALLERY_URL) {
		t.Log(len(page.Data), err)
		if stop {
			break
		}
	}
}

func TestDownloadPagesIter(t *testing.T) {
	stop := false
	go func() {
		<-time.After(time.Second * 15)
		stop = true
	}()
	for page, err := range DownloadPagesIter(TEST_PAGE_URL_0, TEST_PAGE_URL_1) {
		t.Log(len(page.Data), err)
		if stop {
			break
		}
	}
}

func TestBakPageDownload(t *testing.T) {
	ctx, cancel := TimeoutCtx()
	defer cancel()
	// https://e-hentai.org/s/b7a3ead2d6/3138775-24
	downloadPages(ctx, "https://e-hentai.org/s/b7a3ead2d6/3138775-24")
}

func TestJpegPageDownload(t *testing.T) {
	ctx, cancel := TimeoutCtx()
	defer cancel()
	SetCookie("", "", "", "")
	datas, err := downloadPages(ctx, "https://exhentai.org/s/76360befe8/3222212-1")
	if err != nil {
		t.Fatal(err)
	}
	page := datas[0]
	if len(page.Data) == 0 {
		t.Fatal("empty data")
	}
	os.WriteFile("test.jpg", page.Data, 0o644)
}

func TestCache(t *testing.T) {
	// download [TEST_PAGE_URL_0] (write cache)
	// read cache
	// download [TEST_PAGE_URL_0], [TEST_PAGE_URL_1] (write cache)
	// read cache
}
