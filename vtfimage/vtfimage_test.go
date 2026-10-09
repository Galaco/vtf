package vtfimage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/galaco/vtf/v2"
	"github.com/galaco/vtf/v2/format"
)

const samplePath = "../testdata/test.vtf"

// TestConvert_BlockFormats compares against reference images decoded by Pillow (see testdata/bcvectors.py)
func TestConvert_BlockFormats(t *testing.T) {
	tests := []struct {
		name      string
		format    format.Format
		reference string
	}{
		{"DXT1", format.Dxt1, "DXT1"},
		{"DXT1_ONEBITALPHA", format.Dxt1OneBitAlpha, "DXT1"},
		{"DXT3", format.Dxt3, "DXT3"},
		{"DXT5", format.Dxt5, "DXT5"},
		{"ATI1N", format.Ati1n, "ATI1N"},
		{"ATI2N", format.Ati2n, "ATI2N"},
	}
	for _, tc := range tests {
		for _, size := range [][2]int{{16, 16}, {6, 10}} {
			name := filepath.Join("testdata", fmt.Sprintf("%s_%dx%d", tc.reference, size[0], size[1]))
			t.Run(tc.name+"_"+filepath.Base(name), func(t *testing.T) {
				data := readFile(t, name+".bin")
				img, err := Convert(data, size[0], size[1], tc.format)
				if err != nil {
					t.Fatal(err)
				}
				if img.Bounds() != image.Rect(0, 0, size[0], size[1]) {
					t.Fatalf("bounds: got %v", img.Bounds())
				}

				var got, want []byte
				switch img := img.(type) {
				case *image.NRGBA:
					got = img.Pix
					if tc.format == format.Ati2n {
						want = rgbToRGBA(readFile(t, name+".rgb"))
					} else {
						want = readFile(t, name+".rgba")
					}
				case *image.Gray:
					got, want = img.Pix, readFile(t, name+".l")
				default:
					t.Fatalf("unexpected image type %T", img)
				}
				// DXT1 is opaque, where the reference decodes 1 bit alpha
				if tc.format == format.Dxt1 {
					for i := 3; i < len(want); i += 4 {
						want[i] = 255
					}
				}
				if !bytes.Equal(got, want) {
					t.Errorf("decoded pixels do not match reference\ngot  %v\nwant %v", got[:32], want[:32])
				}
			})
		}
	}
}

func TestConvert_DXT1Alpha(t *testing.T) {
	// c0 < c1 selects 3 colour mode, where index 3 is transparent black; every pixel uses index 3
	block := []byte{0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	opaque, _ := Convert(block, 4, 4, format.Dxt1)
	if c := opaque.(*image.NRGBA).NRGBAAt(0, 0); c != (color.NRGBA{0, 0, 0, 255}) {
		t.Errorf("DXT1: got %v, want opaque black", c)
	}
	alpha, _ := Convert(block, 4, 4, format.Dxt1OneBitAlpha)
	if c := alpha.(*image.NRGBA).NRGBAAt(0, 0); c != (color.NRGBA{0, 0, 0, 0}) {
		t.Errorf("DXT1_ONEBITALPHA: got %v, want transparent", c)
	}
}

func TestConvert_Packed(t *testing.T) {
	tests := []struct {
		format format.Format
		pixel  []byte
		want   color.NRGBA
	}{
		{format.RGBA8888, []byte{1, 2, 3, 4}, color.NRGBA{1, 2, 3, 4}},
		{format.ABGR8888, []byte{4, 3, 2, 1}, color.NRGBA{1, 2, 3, 4}},
		{format.RGB888, []byte{1, 2, 3}, color.NRGBA{1, 2, 3, 255}},
		{format.BGR888, []byte{3, 2, 1}, color.NRGBA{1, 2, 3, 255}},
		{format.ARGB8888, []byte{4, 1, 2, 3}, color.NRGBA{1, 2, 3, 4}},
		{format.BGRA8888, []byte{3, 2, 1, 4}, color.NRGBA{1, 2, 3, 4}},
		{format.BGRX8888, []byte{3, 2, 1, 4}, color.NRGBA{1, 2, 3, 255}},
		{format.IA88, []byte{7, 9}, color.NRGBA{7, 7, 7, 9}},
		{format.RGB888BLUESCREEN, []byte{1, 2, 3}, color.NRGBA{1, 2, 3, 255}},
		{format.RGB888BLUESCREEN, []byte{0, 0, 255}, color.NRGBA{0, 0, 255, 0}},
		{format.BGR888BLUESCREEN, []byte{255, 0, 0}, color.NRGBA{0, 0, 255, 0}},
		// Red in the least significant 5 bits
		{format.RGB565, le16(0x001F), color.NRGBA{255, 0, 0, 255}},
		{format.RGB565, le16(0x07E0), color.NRGBA{0, 255, 0, 255}},
		{format.RGB565, le16(0xF800), color.NRGBA{0, 0, 255, 255}},
		{format.RGB565, le16(0x0010), color.NRGBA{132, 0, 0, 255}},
		// Blue in the least significant 5 bits
		{format.BGR565, le16(0x001F), color.NRGBA{0, 0, 255, 255}},
		{format.BGR565, le16(0xF800), color.NRGBA{255, 0, 0, 255}},
		{format.BGRX5551, le16(0x7C00 | 0x8000), color.NRGBA{255, 0, 0, 255}},
		{format.BGRA5551, le16(0x03E0), color.NRGBA{0, 255, 0, 0}},
		{format.BGRA5551, le16(0x801F), color.NRGBA{0, 0, 255, 255}},
		{format.BGRA4444, le16(0x48F2), color.NRGBA{0x88, 0xFF, 0x22, 0x44}},
	}
	for _, tc := range tests {
		img, err := Convert(tc.pixel, 1, 1, tc.format)
		if err != nil {
			t.Errorf("%s: %v", tc.format, err)
			continue
		}
		if got := img.(*image.NRGBA).NRGBAAt(0, 0); got != tc.want {
			t.Errorf("%s % x: got %v, want %v", tc.format, tc.pixel, got, tc.want)
		}
	}
}

func TestConvert_Single(t *testing.T) {
	gray, _ := Convert([]byte{1, 2, 3, 4}, 2, 2, format.I8)
	if got := gray.(*image.Gray).GrayAt(1, 1); got.Y != 4 {
		t.Errorf("I8: got %v, want 4", got)
	}
	alpha, _ := Convert([]byte{1, 2, 3, 4}, 2, 2, format.A8)
	if got := alpha.(*image.Alpha).AlphaAt(1, 0); got.A != 2 {
		t.Errorf("A8: got %v, want 2", got)
	}
}

func TestConvert_Wide(t *testing.T) {
	rgba16 := append(append(le16(0x1234), le16(0xFFFF)...), append(le16(0), le16(0x8000)...)...)
	img, _ := Convert(rgba16, 1, 1, format.RGBA16161616)
	if got := img.(*image.NRGBA64).NRGBA64At(0, 0); got != (color.NRGBA64{0x1234, 0xFFFF, 0, 0x8000}) {
		t.Errorf("RGBA16161616: got %v", got)
	}

	// 0.5, 1, -2 (clamped to 0), +Inf (clamped to 1)
	half := append(append(le16(0x3800), le16(0x3C00)...), append(le16(0xC000), le16(0x7C00)...)...)
	img, _ = Convert(half, 1, 1, format.RGBA16161616F)
	if got := img.(*image.NRGBA64).NRGBA64At(0, 0); got != (color.NRGBA64{0x8000, 0xFFFF, 0, 0xFFFF}) {
		t.Errorf("RGBA16161616F: got %v", got)
	}

	floats := le32f(0.25, 2, float32(math.NaN()))
	img, _ = Convert(floats, 1, 1, format.RGB323232F)
	if got := img.(*image.NRGBA64).NRGBA64At(0, 0); got != (color.NRGBA64{0x4000, 0xFFFF, 0, 0xFFFF}) {
		t.Errorf("RGB323232F: got %v", got)
	}
	img, _ = Convert(le32f(0.5, 0, 1, 0.25), 1, 1, format.RGBA32323232F)
	if got := img.(*image.NRGBA64).NRGBA64At(0, 0); got != (color.NRGBA64{0x8000, 0, 0xFFFF, 0x4000}) {
		t.Errorf("RGBA32323232F: got %v", got)
	}
	img, _ = Convert(le32f(0.5), 1, 1, format.R32F)
	if got := img.(*image.Gray16).Gray16At(0, 0); got.Y != 0x8000 {
		t.Errorf("R32F: got %v", got)
	}
}

func TestHalfToFloat32(t *testing.T) {
	for h := 0; h < 1<<16; h++ {
		sign := 1.0
		if h&0x8000 != 0 {
			sign = -1
		}
		exponent, mantissa := (h>>10)&0x1F, float64(h&0x3FF)
		var want float64
		switch exponent {
		case 0:
			want = sign * math.Ldexp(mantissa, -24)
		case 0x1F:
			if mantissa != 0 {
				if got := halfToFloat32(uint16(h)); !math.IsNaN(float64(got)) {
					t.Errorf("%#04x: got %v, want NaN", h, got)
				}
				continue
			}
			want = math.Inf(int(sign))
		default:
			want = sign * math.Ldexp(1+mantissa/1024, exponent-15)
		}
		if got := halfToFloat32(uint16(h)); float64(got) != want {
			t.Errorf("%#04x: got %v, want %v", h, got, want)
		}
	}
}

func TestConvert_Errors(t *testing.T) {
	for _, f := range []format.Format{format.P8, format.UV88, format.UVWQ8888, format.UVLX8888, format.NvDst16, format.None} {
		if _, err := Convert(make([]byte, 64), 4, 4, f); !errors.Is(err, ErrUnsupportedFormat) {
			t.Errorf("%s: expected %v, got %v", f, ErrUnsupportedFormat, err)
		}
	}
	if _, err := Convert(make([]byte, 15), 2, 2, format.RGBA8888); !errors.Is(err, ErrShortData) {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}
	if _, err := Convert(make([]byte, 7), 1, 1, format.Dxt1); !errors.Is(err, ErrShortData) {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}
	if _, err := Convert(nil, 0, 4, format.RGBA8888); err == nil {
		t.Error("expected error for zero width")
	}
}

func TestDecode(t *testing.T) {
	raw := readFile(t, samplePath)

	img, name, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if name != "vtf" {
		t.Errorf("format name: got %q, want vtf", name)
	}
	if img.Bounds() != image.Rect(0, 0, 512, 128) {
		t.Errorf("bounds: got %v", img.Bounds())
	}
	// BGR888 pixels follow the 80 byte header and 32 byte thumbnail
	for _, p := range []image.Point{{0, 0}, {511, 0}, {100, 127}} {
		offset := 112 + (p.Y*512+p.X)*3
		want := color.NRGBA{raw[offset+2], raw[offset+1], raw[offset], 255}
		if got := img.(*image.NRGBA).NRGBAAt(p.X, p.Y); got != want {
			t.Errorf("pixel %v: got %v, want %v", p, got, want)
		}
	}
}

func TestDecodeConfig(t *testing.T) {
	config, name, err := image.DecodeConfig(bytes.NewReader(readFile(t, samplePath)))
	if err != nil {
		t.Fatal(err)
	}
	if name != "vtf" || config.Width != 512 || config.Height != 128 || config.ColorModel != color.NRGBAModel {
		t.Errorf("got %q %+v", name, config)
	}
}

func TestMipmapAndLowResImage(t *testing.T) {
	texture, err := vtf.ReadFromFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}

	img, err := Mipmap(texture, 0, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 512 {
		t.Errorf("mipmap width: got %d", img.Bounds().Dx())
	}
	if _, err := Mipmap(texture, 1, 0, 0, 0); !errors.Is(err, vtf.ErrOutOfRange) {
		t.Errorf("expected %v, got %v", vtf.ErrOutOfRange, err)
	}

	thumbnail, err := LowResImage(texture)
	if err != nil {
		t.Fatal(err)
	}
	if thumbnail.Bounds() != image.Rect(0, 0, 16, 4) {
		t.Errorf("thumbnail bounds: got %v", thumbnail.Bounds())
	}
}

func FuzzConvert(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, uint8(4), uint8(4), uint32(format.Dxt1))
	f.Fuzz(func(t *testing.T, data []byte, width, height uint8, f uint32) {
		img, err := Convert(data, int(width), int(height), format.Format(f))
		if err == nil && img.Bounds() != image.Rect(0, 0, int(width), int(height)) {
			t.Fatalf("bounds: got %v", img.Bounds())
		}
	})
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func rgbToRGBA(rgb []byte) []byte {
	rgba := make([]byte, 0, len(rgb)/3*4)
	for i := 0; i < len(rgb); i += 3 {
		rgba = append(rgba, rgb[i], rgb[i+1], rgb[i+2], 255)
	}
	return rgba
}

func le16(v uint16) []byte {
	return binary.LittleEndian.AppendUint16(nil, v)
}

func le32f(values ...float32) []byte {
	var b []byte
	for _, v := range values {
		b = binary.LittleEndian.AppendUint32(b, math.Float32bits(v))
	}
	return b
}
