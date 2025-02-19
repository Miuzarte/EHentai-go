package EHentai

import (
	"os"
	"testing"
)

func TestEhQueryFSearch(t *testing.T) {
	// results, err := queryFSearch(EHENTAI_URL, "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	// if err != nil {
	// 	t.Error(err)
	// 	t.FailNow()
	// }
	// t.Logf("%+v", results)

	SetCookie("", "", "", "")
	_, results, err := queryFSearch(EXHENTAI_URL, "耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Logf("%+v", results)
}

func TestBakPageDownload(t *testing.T) {
	// https://e-hentai.org/s/b7a3ead2d6/3138775-24
	downloadPages("https://e-hentai.org/s/b7a3ead2d6/3138775-24")
}

func TestJpegPageDownload(t *testing.T) {
	SetCookie("", "", "", "")
	datas, err := downloadPages("https://exhentai.org/s/76360befe8/3222212-1")
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
