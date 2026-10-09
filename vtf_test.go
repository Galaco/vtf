package vtf

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/galaco/vtf/format"
)

const samplePath = "testdata/test.vtf"

// loadSample returns the raw bytes of the sample texture: v7.2 BGR888 512x128, 1 mipmap, 16x4 thumbnail
func loadSample(t testing.TB) ([]byte, *Vtf) {
	t.Helper()

	raw, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ReadFromStream(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return raw, v
}

func TestVtf_Header(t *testing.T) {
	_, v := loadSample(t)

	h := v.Header()
	if h.Signature != [4]byte{'V', 'T', 'F', 0} {
		t.Error("format signature mismatch")
	}
	if h.Version != [2]uint32{7, 2} {
		t.Errorf("version: got %v, want 7.2", h.Version)
	}
	if h.HeaderSize != 80 {
		t.Errorf("header size: got %d, want 80", h.HeaderSize)
	}
	if h.Width != 512 || h.Height != 128 || h.Depth != 1 {
		t.Errorf("dimensions: got %dx%dx%d, want 512x128x1", h.Width, h.Height, h.Depth)
	}
	if h.Flags != FlagNoMipmaps|FlagNoLevelOfDetail {
		t.Errorf("flags: got %#x, want %#x", h.Flags, FlagNoMipmaps|FlagNoLevelOfDetail)
	}
	if h.Frames != 1 || h.MipmapCount != 1 {
		t.Errorf("got %d frames and %d mipmaps, want 1 and 1", h.Frames, h.MipmapCount)
	}
	if format.Format(h.HighResImageFormat) != format.BGR888 {
		t.Errorf("high res format: got %s, want BGR888", format.Format(h.HighResImageFormat))
	}
	if format.Format(h.LowResImageFormat) != format.Dxt1 || h.LowResImageWidth != 16 || h.LowResImageHeight != 4 {
		t.Errorf("low res: got %s %dx%d, want DXT1 16x4",
			format.Format(h.LowResImageFormat), h.LowResImageWidth, h.LowResImageHeight)
	}
	if h.NumResource != 0 {
		t.Errorf("resources: got %d, want 0 for a 7.2 texture", h.NumResource)
	}
}

func TestVtf_HighResImageData(t *testing.T) {
	raw, v := loadSample(t)

	mipmaps := v.HighResImageData()
	if len(mipmaps) != 1 || len(mipmaps[0]) != 1 || len(mipmaps[0][0]) != 1 || len(mipmaps[0][0][0]) != 1 {
		t.Fatal("expected a single mipmap, frame, face and slice")
	}
	// Header (80) + DXT1 16x4 thumbnail (32)
	if !bytes.Equal(mipmaps[0][0][0][0], raw[112:]) {
		t.Error("high resolution data does not match file contents")
	}
}

func TestVtf_LowResImageData(t *testing.T) {
	raw, v := loadSample(t)

	if !bytes.Equal(v.LowResImageData(), raw[80:112]) {
		t.Error("low resolution data does not match file contents")
	}
}

func TestVtf_Image(t *testing.T) {
	tt := newTestTexture()
	tt.frames = 3
	data, images := tt.build(t)
	v, err := ReadFromStream(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	assertImages(t, v, images)

	largest := func(frame int) []byte {
		for _, img := range images {
			if img.level == 0 && img.frame == frame {
				return bytes.Repeat([]byte{img.fill}, img.size)
			}
		}
		t.Fatalf("no image for frame %d", frame)
		return nil
	}

	if !bytes.Equal(v.Image(), largest(0)) {
		t.Error("Image() did not return the largest mipmap of the first frame")
	}
	if !bytes.Equal(v.HighestResolutionImageForFrame(2), largest(2)) {
		t.Error("HighestResolutionImageForFrame(2) did not return the largest mipmap of frame 2")
	}

	mipmaps := v.MipmapsForFrame(1)
	if len(mipmaps) != int(tt.mipmaps) {
		t.Fatalf("got %d mipmaps, want %d", len(mipmaps), tt.mipmaps)
	}
	for _, img := range images {
		if img.frame != 1 {
			continue
		}
		// Mipmaps are indexed smallest first
		if got := mipmaps[len(mipmaps)-1-img.level]; !bytes.Equal(got, bytes.Repeat([]byte{img.fill}, img.size)) {
			t.Errorf("mipmap %d of frame 1 does not match", img.level)
		}
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
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "zero height",
			modifyHeader: func(h *Header) {
				h.Height = 0
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "excessive width",
			modifyHeader: func(h *Header) {
				h.Width = 20000
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "excessive height",
			modifyHeader: func(h *Header) {
				h.Height = 20000
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "zero mipmap count",
			modifyHeader: func(h *Header) {
				h.MipmapCount = 0
			},
			expectedError: ErrInvalidMipmapCount,
		},
		{
			name: "excessive mipmap count",
			modifyHeader: func(h *Header) {
				h.MipmapCount = 100
			},
			expectedError: ErrInvalidMipmapCount,
		},
		{
			name: "zero frame count",
			modifyHeader: func(h *Header) {
				h.Frames = 0
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "excessive frame count",
			modifyHeader: func(h *Header) {
				h.Frames = 1025
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "unsupported major version",
			modifyHeader: func(h *Header) {
				h.Version[0] = 6
				h.Version[1] = 0
			},
			expectedError: ErrUnsupportedVersion,
		},
		{
			name: "unsupported minor version",
			modifyHeader: func(h *Header) {
				h.Version[1] = 6
			},
			expectedError: ErrUnsupportedVersion,
		},
		{
			name: "invalid header size",
			modifyHeader: func(h *Header) {
				h.HeaderSize = 10
			},
			expectedError: ErrInvalidHeaderSize,
		},
		{
			name: "excessive header size",
			modifyHeader: func(h *Header) {
				h.HeaderSize = 2048
			},
			expectedError: ErrInvalidHeaderSize,
		},
		{
			name: "excessive low res dimensions",
			modifyHeader: func(h *Header) {
				h.LowResImageWidth = 32
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "excessive depth",
			modifyHeader: func(h *Header) {
				h.Depth = 20000
			},
			expectedError: ErrInvalidDimensions,
		},
		{
			name: "excessive mipmap count for depth",
			modifyHeader: func(h *Header) {
				h.Width, h.Height, h.Depth = 4, 4, 16
				h.MipmapCount = 6
			},
			expectedError: ErrInvalidMipmapCount,
		},
		{
			name: "excessive resource count",
			modifyHeader: func(h *Header) {
				h.Version[1] = 3
				h.NumResource = 33
				h.HeaderSize = 1000
			},
			expectedError: ErrInvalidResource,
		},
		{
			name: "resources exceed header size",
			modifyHeader: func(h *Header) {
				h.Version[1] = 3
				h.NumResource = 3
				h.HeaderSize = 96
			},
			expectedError: ErrInvalidHeaderSize,
		},
	}

	// Load a valid VTF file first to get a valid header
	_, vtf := loadSample(t)

	reader := &Reader{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with valid header
			header := vtf.Header()
			// Modify it to be invalid
			tt.modifyHeader(&header)
			// Validate should fail
			err := reader.validateHeader(&header)
			if err == nil {
				t.Fatal("expected error but got none")
			}
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}
