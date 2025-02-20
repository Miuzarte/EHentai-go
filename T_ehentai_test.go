package EHentai

import (
	"testing"
	"time"
)

func TestFetchGalleryPageUrls(t *testing.T) {
	tn := time.Now()
	t.Log(FetchGalleryPageUrls("https://e-hentai.org/g/3138775/30b0285f9b/"))
	t.Log(time.Since(tn))
}

func TestFetchGalleryImageUrls(t *testing.T) {
	tn := time.Now()
	t.Log(FetchGalleryImageUrls("https://e-hentai.org/g/3138775/30b0285f9b/"))
	t.Log(time.Since(tn))
}

func TestFetchPageImageUrl(t *testing.T) {
	tn := time.Now()
	t.Log(FetchPageImageUrl("https://e-hentai.org/s/0b2127ea05/3138775-8"))
	t.Log(time.Since(tn))
}

func TestDownloadPageImage(t *testing.T) {
	img, err := DownloadPages("https://e-hentai.org/s/0b2127ea05/3138775-8")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log(len(img))
}
