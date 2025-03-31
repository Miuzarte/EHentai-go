package EHentai

import (
	"testing"
)

func TestEhTagInit(t *testing.T) {
	err := ehTagDatabase.Init()
	if err != nil {
		t.Fatal(err)
	}
	if len(ehTagDatabase) == 0 {
		t.Fatal("empty database")
	}
	if len(ehTagDatabase["rows"]) == 0 {
		t.Fatal("empty rows")
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
