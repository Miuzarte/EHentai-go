package EHentai

import (
	"testing"
)

func TestFetchGalleryImageUrls(t *testing.T) {
	t.Log(FetchGalleryImageUrls("https://e-hentai.org/g/3138775/30b0285f9b/"))
}

func TestFetchPageImageUrl(t *testing.T) {
	t.Log(FetchPageImageUrl("https://e-hentai.org/s/0b2127ea05/3138775-8"))
}

func TestDownloadPageImage(t *testing.T) {
	img, err := DownloadPages("https://e-hentai.org/s/0b2127ea05/3138775-8")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log(len(img))
}
