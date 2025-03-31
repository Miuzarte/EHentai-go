package EHentai

import (
	"testing"
)

func TestDatabaseInit(t *testing.T) {
	err := ehTagDatabase.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ehTagDatabase.Info())
	t.Log(ehTagDatabase["rows"])
}

func TestTranslate(t *testing.T) {
	ehTagDatabase.Init()
	t.Log(Translate("language:chinese"))
	t.Log(Translate("language:translated"))
	t.Log(Translate("parody:original"))
	t.Log(Translate("female:cunnilingus"))
	t.Log(Translate("female:females only"))
	t.Log(Translate("female:fingering"))
	t.Log(Translate("female:masturbation"))
	t.Log(Translate("female:mesuiki"))
	t.Log(Translate("female:sex toys"))
	t.Log(Translate("female:squirting"))
	t.Log(Translate("female:tribadism"))
	t.Log(Translate("female:yuri"))
}
