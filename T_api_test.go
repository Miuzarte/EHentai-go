package EHentai

import "testing"

func TestPostGalleryMetadata(t *testing.T) {
	resp, err := PostGalleryMetadata(GIdList{3138775, "30b0285f9b"})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Logf("%+v", *resp)
}

func TestPostGalleryToken(t *testing.T) {
	resp, err := PostGalleryToken(PageList{"0b2127ea05", 3138775, 8})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Logf("%+v", *resp)
}

func TestPostGalleryMetadataExhResults(t *testing.T) {
	resp, err := PostGalleryMetadata(
		GIdList{3239359, "ed3b4ef37a"},
		GIdList{3138775, "30b0285f9b"},
		GIdList{3138763, "01c4c22a37"}, // from exhentai but still works
		GIdList{3121784, "92adae3fa6"}, // from exhentai but still works
	)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Logf("%+v", *resp)
}
