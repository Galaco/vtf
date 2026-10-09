package internal

import (
	"errors"
	"testing"

	"github.com/galaco/vtf/format"
)

func TestMipmapDimensions(t *testing.T) {
	want := [][3]int{
		{512, 128, 4}, {256, 64, 2}, {128, 32, 1}, {64, 16, 1}, {32, 8, 1},
		{16, 4, 1}, {8, 2, 1}, {4, 1, 1}, {2, 1, 1}, {1, 1, 1}, {1, 1, 1},
	}
	for level, dims := range want {
		w, h, d := MipmapDimensions(512, 128, 4, level)
		if [3]int{w, h, d} != dims {
			t.Errorf("level %d: got %dx%dx%d, want %v", level, w, h, d, dims)
		}
	}
}

func TestComputeSizeOfMipmapData(t *testing.T) {
	// Expected sizes for 1x1, 2x2, 4x4, 8x2 and 16x16 images
	tests := []struct {
		format format.Format
		sizes  [5]int
	}{
		{format.RGBA8888, [5]int{4, 16, 64, 64, 1024}},
		{format.ABGR8888, [5]int{4, 16, 64, 64, 1024}},
		{format.RGB888, [5]int{3, 12, 48, 48, 768}},
		{format.BGR888, [5]int{3, 12, 48, 48, 768}},
		{format.RGB565, [5]int{2, 8, 32, 32, 512}},
		{format.I8, [5]int{1, 4, 16, 16, 256}},
		{format.IA88, [5]int{2, 8, 32, 32, 512}},
		{format.P8, [5]int{1, 4, 16, 16, 256}},
		{format.A8, [5]int{1, 4, 16, 16, 256}},
		{format.RGB888BLUESCREEN, [5]int{3, 12, 48, 48, 768}},
		{format.BGR888BLUESCREEN, [5]int{3, 12, 48, 48, 768}},
		{format.ARGB8888, [5]int{4, 16, 64, 64, 1024}},
		{format.BGRA8888, [5]int{4, 16, 64, 64, 1024}},
		{format.Dxt1, [5]int{8, 8, 8, 16, 128}},
		{format.Dxt3, [5]int{16, 16, 16, 32, 256}},
		{format.Dxt5, [5]int{16, 16, 16, 32, 256}},
		{format.BGRX8888, [5]int{4, 16, 64, 64, 1024}},
		{format.BGR565, [5]int{2, 8, 32, 32, 512}},
		{format.BGRX5551, [5]int{2, 8, 32, 32, 512}},
		{format.BGRA4444, [5]int{2, 8, 32, 32, 512}},
		{format.Dxt1OneBitAlpha, [5]int{8, 8, 8, 16, 128}},
		{format.BGRA5551, [5]int{2, 8, 32, 32, 512}},
		{format.UV88, [5]int{2, 8, 32, 32, 512}},
		{format.UVWQ8888, [5]int{4, 16, 64, 64, 1024}},
		{format.RGBA16161616F, [5]int{8, 32, 128, 128, 2048}},
		{format.RGBA16161616, [5]int{8, 32, 128, 128, 2048}},
		{format.UVLX8888, [5]int{4, 16, 64, 64, 1024}},
		{format.R32F, [5]int{4, 16, 64, 64, 1024}},
		{format.RGB323232F, [5]int{12, 48, 192, 192, 3072}},
		{format.RGBA32323232F, [5]int{16, 64, 256, 256, 4096}},
		{format.Ati2n, [5]int{16, 16, 16, 32, 256}},
		{format.Ati1n, [5]int{8, 8, 8, 16, 128}},
	}
	dims := [5][2]int{{1, 1}, {2, 2}, {4, 4}, {8, 2}, {16, 16}}

	for _, tt := range tests {
		for i, d := range dims {
			got, err := ComputeSizeOfMipmapData(d[0], d[1], tt.format)
			if err != nil {
				t.Errorf("%s %dx%d: unexpected error: %v", tt.format, d[0], d[1], err)
			}
			if got != tt.sizes[i] {
				t.Errorf("%s %dx%d: got %d bytes, want %d", tt.format, d[0], d[1], got, tt.sizes[i])
			}
		}
	}
}

func TestComputeSizeOfMipmapData_EveryFormat(t *testing.T) {
	supported := map[format.Format]bool{
		format.NvDst16: false, format.NvDst24: false, format.NvIntz: false, format.NvRawz: false,
		format.AtiDst16: false, format.AtiDst24: false, format.NvNull: false,
	}
	for f := format.RGBA8888; f <= format.Ati1n; f++ {
		size, err := ComputeSizeOfMipmapData(4, 4, f)
		if want, ok := supported[f]; ok && !want {
			if !errors.Is(err, ErrUnsupportedFormat) {
				t.Errorf("%s: expected ErrUnsupportedFormat, got %v", f, err)
			}
			continue
		}
		if err != nil || size == 0 {
			t.Errorf("%s: got size %d, error %v", f, size, err)
		}
	}
	for _, f := range []format.Format{format.None, format.Ati1n + 1} {
		if _, err := ComputeSizeOfMipmapData(4, 4, f); !errors.Is(err, ErrUnsupportedFormat) {
			t.Errorf("%s: expected ErrUnsupportedFormat, got %v", f, err)
		}
	}
}

func TestComputeSizeOfMipmapData_Empty(t *testing.T) {
	for _, f := range []format.Format{format.Dxt1, format.RGBA8888} {
		if size, err := ComputeSizeOfMipmapData(0, 0, f); size != 0 || err != nil {
			t.Errorf("%s 0x0: got %d bytes, error %v", f, size, err)
		}
	}
}
