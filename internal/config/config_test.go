package config

import (
	"bytes"
	"os"
	"testing"
)

func TestGameConfig(t *testing.T) {
	dir := os.Getenv("LOLDATA")
	if dir == "" {
		dir = "../../game"
	}
	b, err := os.ReadFile(dir + "/" + FileName)
	if err != nil {
		t.Skip("no LANDS.CFG; set LOLDATA")
	}
	c, ok := Decode(b)
	if !ok || !c.VideoSupported() {
		t.Fatalf("%+v ok=%v", c, ok)
	}
	if !bytes.Equal(c.Encode(), b) {
		t.Fatalf("round trip: % x vs % x", c.Encode(), b)
	}
	b[0]++
	if _, ok := Decode(b); ok {
		t.Fatal("bad checksum accepted")
	}
}
