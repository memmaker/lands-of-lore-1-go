package data

import (
	"strings"
	"testing"
)

func TestLevelStrings(t *testing.T) {
	f, err := Open(gameDir(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.Read("LEVEL01.ENG")
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for i := range 5 {
		all = append(all, string(Unescape(Unpack(TableString(b, i)))))
	}
	text := strings.Join(all, "|")
	t.Log(text)
	if !strings.Contains(text, "the") && !strings.Contains(text, "The") {
		t.Fatalf("no English in %q", text)
	}
}
