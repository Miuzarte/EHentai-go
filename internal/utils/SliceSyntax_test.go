package utils

import (
	"fmt"
	"testing"
)

func TestSliceSyntaxes(t *testing.T) {
	const s = "[:1], [1:5][5:8]|[12:8]"
	const l = 12

	sss, err := ParseSliceSyntaxes(s)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(sss.String())

	indexes, err := sss.ToIndexes(l)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(indexes)

	indexes, err = sss.ToIndexesNoRepeat(l)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(indexes)
}
