package vtf

import (
	"errors"
	"io/fs"
	"os"
	"testing"
)

func TestReadFromFile(t *testing.T) {
	v, err := ReadFromFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	if v.Header().Width != 512 {
		t.Errorf("width: got %d, want 512", v.Header().Width)
	}
}

func TestReadFromFile_Missing(t *testing.T) {
	_, err := ReadFromFile("testdata/missing.vtf")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestReadFromStream(t *testing.T) {
	f, err := os.Open(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err = ReadFromStream(f); err != nil {
		t.Fatal(err)
	}
}
