package vtf

import (
	"os"
	"testing"
)

func TestVtf_Header(t *testing.T) {
	f, err := os.Open("samples/read/test.vtf")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	vtf, err := ReadFromStream(f)
	if err != nil {
		t.Error(err)
	}
	if vtf.Header().Signature != [4]byte{86, 84, 70, 0} {
		t.Error("format signature mismatch")
	}
}

func TestVtf_HighResImageData(t *testing.T) {
	f, err := os.Open("samples/read/test.vtf")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	vtf, err := ReadFromStream(f)
	if err != nil {
		t.Error(err)
	}

	if len(vtf.HighResImageData()) == 0 {
		t.Error("expected high resolution data, received none")
	}
}

func TestVtf_LowResImageData(t *testing.T) {
	f, err := os.Open("samples/read/test.vtf")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	vtf, err := ReadFromStream(f)
	if err != nil {
		t.Error(err)
	}

	if len(vtf.LowResImageData()) == 0 {
		t.Error("expected low resolution data, received none")
	}
}
