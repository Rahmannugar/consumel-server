package openapi

import (
	"bytes"
	"os"
	"testing"
)

func TestGeneratedDocumentIsCurrent(t *testing.T) {
	want, err := Document()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("openapi.json is stale; run task generate")
	}
}
