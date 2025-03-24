package EHentai

import (
	"os"
	"testing"
	"time"
)

func TestEhQueryFSearch(t *testing.T) {
	// results, err := queryFSearch(EHENTAI_URL, "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	// if err != nil {
	// 	t.Error(err)
	// 	t.FailNow()
	// }
	// t.Logf("%+v", results)

	SetCookie("", "", "", "")
	_, results, err := querySearch(EXHENTAI_URL, "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Logf("%+v", results)
}

func TestFetchGalleryPageUrls(t *testing.T) {
	t.Log(FetchGalleryPageUrls("https://e-hentai.org/g/3138775/30b0285f9b/"))
}

func TestFetchGalleryImageUrls(t *testing.T) {
	t.Log(FetchGalleryImageUrls("https://e-hentai.org/g/3138775/30b0285f9b/"))
}

func TestFetchPageImageUrl(t *testing.T) {
	t.Log(FetchPageImageUrl("https://e-hentai.org/s/0b2127ea05/3138775-8"))
}

func TestDownloadPages(t *testing.T) {
	ctx, cancel := TimeoutCtx()
	defer cancel()
	img, err := DownloadPages(ctx, "https://e-hentai.org/s/0b2127ea05/3138775-8")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log(len(img[0]))
}

func TestDownlaodGalleryIter(t *testing.T) {
	it, err := DownlaodGalleryIter("https://e-hentai.org/g/3138775/30b0285f9b/")
	if err != nil {
		t.Fatal(err)
	}
	stop := false
	go func() {
		<-time.After(time.Second * 15)
		stop = true
	}()
	for data, err := range it {
		t.Log(len(data), err)
		if stop {
			break
		}
	}
}

func TestDownloadPagesIter(t *testing.T) {
	it, err := DownloadPagesIter("https://e-hentai.org/s/0b2127ea05/3138775-8")
	if err != nil {
		t.Fatal(err)
	}
	stop := false
	go func() {
		<-time.After(time.Second * 15)
		stop = true
	}()
	for data, err := range it {
		t.Log(len(data), err)
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
		t.Error(err)
		t.FailNow()
	}
	data := datas[0]
	if len(data) == 0 {
		t.Error("empty data")
		t.FailNow()
	}
	os.WriteFile("test.jpg", data, 0o644)
}
