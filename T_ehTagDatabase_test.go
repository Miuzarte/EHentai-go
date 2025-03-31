package EHentai

import (
	"testing"
)

func TestEhTagInit(t *testing.T) {
	err := ehTagDatabase.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ehTagDatabase.Info())
	t.Log(ehTagDatabase["rows"])
}

func TestEhTagTranslate(t *testing.T) {
	if !ehTagDatabase.Ok() {
		ehTagDatabase.Init()
	}
	tags := []string{
		"language:chinese",
		"language:translated",
		"parody:original",
		"female:cunnilingus",
		"female:females only",
		"female:fingering",
		"female:masturbation",
		"female:mesuiki",
		"female:sex toys",
		"female:squirting",
		"female:tribadism",
		"female:yuri",
	}
	for _, v := range tags {
		t.Log(Translate(v))
	}
}
