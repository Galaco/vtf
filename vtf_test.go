package vtf

import (
	"errors"
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

func TestVtf_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		modifyHeader  func(*Header)
		expectedError error
	}{
		{
			name: "zero width",
			modifyHeader: func(h *Header) {
				h.Width = 0
			},
			expectedError: ErrorInvalidDimensions,
		},
		{
			name: "zero height",
			modifyHeader: func(h *Header) {
				h.Height = 0
			},
			expectedError: ErrorInvalidDimensions,
		},
		{
			name: "excessive width",
			modifyHeader: func(h *Header) {
				h.Width = 20000
			},
			expectedError: ErrorInvalidDimensions,
		},
		{
			name: "excessive height",
			modifyHeader: func(h *Header) {
				h.Height = 20000
			},
			expectedError: ErrorInvalidDimensions,
		},
		{
			name: "zero mipmap count",
			modifyHeader: func(h *Header) {
				h.MipmapCount = 0
			},
			expectedError: ErrorInvalidMipmapCount,
		},
		{
			name: "excessive mipmap count",
			modifyHeader: func(h *Header) {
				h.MipmapCount = 100
			},
			expectedError: ErrorInvalidMipmapCount,
		},
		{
			name: "zero frame count",
			modifyHeader: func(h *Header) {
				h.Frames = 0
			},
			expectedError: ErrorInvalidDimensions,
		},
		{
			name: "unsupported version",
			modifyHeader: func(h *Header) {
				h.Version[0] = 6
				h.Version[1] = 0
			},
			expectedError: ErrorUnsupportedVersion,
		},
		{
			name: "invalid header size",
			modifyHeader: func(h *Header) {
				h.HeaderSize = 10
			},
			expectedError: ErrorInvalidHeaderSize,
		},
	}

	// Load a valid VTF file first to get a valid header
	f, err := os.Open("samples/read/test.vtf")
	if err != nil {
		t.Fatal(err)
	}
	vtf, err := ReadFromStream(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	reader := &Reader{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with valid header
			header := vtf.Header()
			// Modify it to be invalid
			tt.modifyHeader(&header)
			// Validate should fail
			err := reader.validateHeader(&header, 10000)
			if err == nil {
				t.Errorf("expected error but got none")
			}
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}
