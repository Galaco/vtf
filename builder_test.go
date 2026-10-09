package vtf

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/galaco/vtf/format"
)

// testTexture describes a synthetic vtf to encode in tests. Every stored image is
// filled with a unique byte, so tests can verify each one is decoded from the right place.
type testTexture struct {
	minorVersion  uint32
	width, height uint16
	depth         uint16
	format        format.Format
	mipmaps       uint8
	frames        uint16
	firstFrame    uint16
	flags         uint32
	faces         int // faces to encode
	lowResFormat  format.Format
	lowResWidth   uint8
	lowResHeight  uint8
	trailing      []byte
}

// storedImage is a single mipmap/frame/face/slice image within an encoded testTexture.
// Level 0 is the largest mipmap.
type storedImage struct {
	level, frame, face, slice int
	fill                      byte
	size                      int
}

func newTestTexture() testTexture {
	return testTexture{
		minorVersion: 2,
		width:        16,
		height:       16,
		depth:        1,
		format:       format.RGBA8888,
		mipmaps:      5,
		frames:       1,
		faces:        1,
		lowResFormat: format.Dxt1,
		lowResWidth:  16,
		lowResHeight: 16,
	}
}

const testLowResFill = 0xEE

func (tt testTexture) header() Header {
	h := Header{}
	copy(h.Signature[:], vtfSignature)
	h.Version = [2]uint32{7, tt.minorVersion}
	h.HeaderSize = tt.headerSize()
	h.Width = tt.width
	h.Height = tt.height
	h.Flags = tt.flags
	h.Frames = tt.frames
	h.FirstFrame = tt.firstFrame
	h.Reflectivity = [3]float32{0.25, 0.5, 0.75}
	h.BumpmapScale = 1
	h.HighResImageFormat = uint32(tt.format)
	h.MipmapCount = tt.mipmaps
	h.LowResImageFormat = uint32(tt.lowResFormat)
	h.LowResImageWidth = tt.lowResWidth
	h.LowResImageHeight = tt.lowResHeight
	if tt.minorVersion >= 2 {
		h.Depth = tt.depth
	}
	return h
}

func (tt testTexture) headerSize() uint32 {
	if tt.minorVersion < 2 {
		return 64
	}
	return 80
}

// build encodes the texture, returning the file and the images in the order they are stored.
func (tt testTexture) build(t testing.TB) ([]byte, []storedImage) {
	t.Helper()

	h := tt.header()
	encoded := make([]byte, h.HeaderSize)
	copy(encoded, encodeHeader(t, h))

	if tt.lowResFormat != format.None && tt.lowResWidth > 0 && tt.lowResHeight > 0 {
		lowRes := testImageSize(t, int(tt.lowResWidth), int(tt.lowResHeight), tt.lowResFormat)
		encoded = append(encoded, bytes.Repeat([]byte{testLowResFill}, lowRes)...)
	}

	var images []storedImage
	fill := byte(0)
	for level := int(tt.mipmaps) - 1; level >= 0; level-- {
		width := testMipmapDimension(int(tt.width), level)
		height := testMipmapDimension(int(tt.height), level)
		depth := testMipmapDimension(int(tt.depth), level)
		size := testImageSize(t, width, height, tt.format)
		for frame := 0; frame < int(tt.frames); frame++ {
			for face := 0; face < tt.faces; face++ {
				for slice := 0; slice < depth; slice++ {
					fill = fill%250 + 1
					images = append(images, storedImage{level, frame, face, slice, fill, size})
					encoded = append(encoded, bytes.Repeat([]byte{fill}, size)...)
				}
			}
		}
	}

	return append(encoded, tt.trailing...), images
}

func binaryWrite(w io.Writer, h *Header) error {
	return binary.Write(w, binary.LittleEndian, h)
}

func testMipmapDimension(size, level int) int {
	if size>>level < 1 {
		return 1
	}
	return size >> level
}

// testImageSize is deliberately independent of the internal package, so tests check its maths.
func testImageSize(t testing.TB, width, height int, f format.Format) int {
	t.Helper()

	blocks := ((width + 3) / 4) * ((height + 3) / 4)
	switch f {
	case format.Dxt1, format.Dxt1OneBitAlpha, format.Ati1n:
		return blocks * 8
	case format.Dxt3, format.Dxt5, format.Ati2n:
		return blocks * 16
	case format.I8, format.A8:
		return width * height
	case format.BGR888, format.RGB888:
		return width * height * 3
	case format.RGBA8888, format.BGRA8888:
		return width * height * 4
	case format.RGBA16161616F:
		return width * height * 8
	case format.RGBA32323232F:
		return width * height * 16
	}
	t.Fatalf("test builder does not support format %s", f)
	return 0
}

// assertImages checks every stored image was decoded into the right mipmap/frame/face/slice.
func assertImages(t *testing.T, v *Vtf, images []storedImage) {
	t.Helper()

	mipmaps := v.HighResImageData()
	count := 0
	for _, frames := range mipmaps {
		for _, faces := range frames {
			for _, slices := range faces {
				count += len(slices)
			}
		}
	}
	if count != len(images) {
		t.Errorf("decoded %d images, want %d", count, len(images))
	}

	for _, img := range images {
		// Mipmaps are indexed smallest first
		idx := len(mipmaps) - 1 - img.level
		if idx < 0 || img.frame >= len(mipmaps[idx]) || img.face >= len(mipmaps[idx][img.frame]) ||
			img.slice >= len(mipmaps[idx][img.frame][img.face]) {
			t.Fatalf("missing image for mipmap %d frame %d face %d slice %d", img.level, img.frame, img.face, img.slice)
		}
		got := mipmaps[idx][img.frame][img.face][img.slice]
		if want := bytes.Repeat([]byte{img.fill}, img.size); !bytes.Equal(got, want) {
			t.Errorf("mipmap %d frame %d face %d slice %d: got %d bytes starting %v, want %d bytes of %d",
				img.level, img.frame, img.face, img.slice, len(got), got[:min(len(got), 4)], img.size, img.fill)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
