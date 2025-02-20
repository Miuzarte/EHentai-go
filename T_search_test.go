package EHentai

import "testing"

func TestEHSearch(t *testing.T) {
	_, galleries, err := EHSearch("耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	for _, gallery := range galleries {
		t.Logf("\n%+v\n", gallery)
	}
	_, galleries, err = EHSearch("耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜", CATEGORY_MANGA)
	if err != nil && err != ErrNoHitsFound {
		t.Error(err)
		t.FailNow()
	}
	for _, gallery := range galleries {
		t.Logf("\n%+v\n", gallery)
	}
}

func TestEHSearchTag(t *testing.T) {
	_, galleries, err := EHSearch("chinese")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	for _, gallery := range galleries {
		t.Logf("\n%+v\n", gallery)
	}
}

func TestEHSearchDetail(t *testing.T) {
	_, galleries, err := EHSearchDetail("耳で恋した同僚〜オナサポ音声オタク女が同僚の声に反応してイキまくり〜")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	for _, gallery := range galleries {
		t.Logf("\n%+v\n", gallery)
	}
}
