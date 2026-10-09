package vtf

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/galaco/vtf/v2/format"
)

func TestRead_Formats(t *testing.T) {
	formats := []format.Format{
		format.RGBA8888, format.BGRA8888, format.BGR888, format.I8, format.A8,
		format.RGBA16161616F, format.RGBA32323232F,
		format.Dxt1, format.Dxt1OneBitAlpha, format.Dxt3, format.Dxt5, format.Ati1n, format.Ati2n,
	}
	for _, f := range formats {
		t.Run(f.String(), func(t *testing.T) {
			// Non-square, so the smallest mipmaps are narrower than a 4x4 compression block
			tt := newTestTexture()
			tt.width, tt.height = 32, 8
			tt.mipmaps = 6
			tt.format = f
			data, images := tt.build(t)

			v, err := ReadFromStream(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			assertImages(t, v, images)
		})
	}
}

func TestRead_UnsupportedFormat(t *testing.T) {
	for _, f := range []format.Format{format.NvDst16, format.NvNull, format.Format(99)} {
		t.Run(f.String(), func(t *testing.T) {
			tt := newTestTexture()
			tt.format = format.RGBA8888
			data, _ := tt.build(t)
			// Corrupt the stored format, as the builder can't encode an unsupported one
			h := tt.header()
			h.HighResImageFormat = f
			copy(data, encodeHeader(t, h))

			_, err := ReadFromStream(bytes.NewReader(data))
			if !errors.Is(err, ErrUnsupportedFormat) {
				t.Errorf("expected %v, got %v", ErrUnsupportedFormat, err)
			}
		})
	}
}

func TestRead_NoLowResImage(t *testing.T) {
	cases := map[string]func(*testTexture){
		"zero size":   func(tt *testTexture) { tt.lowResWidth, tt.lowResHeight = 0, 0 },
		"zero width":  func(tt *testTexture) { tt.lowResWidth = 0 },
		"format none": func(tt *testTexture) { tt.lowResFormat = format.None },
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			tt := newTestTexture()
			modify(&tt)
			data, images := tt.build(t)

			v, err := ReadFromStream(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if len(v.LowResImageData()) != 0 {
				t.Errorf("expected no low resolution data, got %d bytes", len(v.LowResImageData()))
			}
			assertImages(t, v, images)
		})
	}
}

func TestRead_LowResImage(t *testing.T) {
	tt := newTestTexture()
	tt.lowResWidth, tt.lowResHeight = 16, 2
	data, images := tt.build(t)

	v, err := ReadFromStream(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	// DXT1 pads to 4x4 blocks: 4x1 blocks of 8 bytes
	if want := bytes.Repeat([]byte{testLowResFill}, 32); !bytes.Equal(v.LowResImageData(), want) {
		t.Errorf("low resolution data: got %d bytes, want %d", len(v.LowResImageData()), len(want))
	}
	assertImages(t, v, images)
}

func TestRead_Versions(t *testing.T) {
	for _, minor := range []uint32{0, 1, 2} {
		tt := newTestTexture()
		tt.minorVersion = minor
		tt.frames = 2
		data, images := tt.build(t)

		v, err := ReadFromStream(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("7.%d: %v", minor, err)
		}
		assertImages(t, v, images)

		h := v.Header()
		if h.HeaderSize != tt.headerSize() || h.Width != tt.width || h.Frames != tt.frames {
			t.Errorf("7.%d: header was not decoded: %+v", minor, h)
		}
		// Depth is only stored from 7.2; fields beyond older headers are not decoded
		if wantDepth := map[uint32]uint16{0: 0, 1: 0, 2: 1}[minor]; h.Depth != wantDepth {
			t.Errorf("7.%d: depth: got %d, want %d", minor, h.Depth, wantDepth)
		}
	}
}

func TestRead_TrailingBytes(t *testing.T) {
	raw, want := loadSample(t)

	data := append(append([]byte{}, raw...), 0xAA, 0xBB, 0xCC)
	v, err := ReadFromStream(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(v.Image(), want.Image()) {
		t.Error("trailing bytes changed the decoded image")
	}
	if !bytes.Equal(v.LowResImageData(), want.LowResImageData()) {
		t.Error("trailing bytes changed the decoded low resolution image")
	}
}

func TestRead_Truncated(t *testing.T) {
	tt := newTestTexture()
	tt.format = format.Dxt5
	tt.frames = 2
	data, _ := tt.build(t)

	// Every possible truncation must be an error, never a panic or partial texture
	for n := 0; n < len(data); n++ {
		if _, err := ReadFromStream(bytes.NewReader(data[:n])); err == nil {
			t.Fatalf("truncated to %d of %d bytes: expected error", n, len(data))
		}
	}

	raw, _ := loadSample(t)
	for _, n := range []int{0, 15, 16, 79, 80, 100, 111, 112, 1000, len(raw) - 1} {
		_, err := ReadFromStream(bytes.NewReader(raw[:n]))
		switch {
		case n < 80 && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, ErrInvalidHeaderSize):
			t.Errorf("sample truncated to %d bytes: expected truncated header error, got %v", n, err)
		case n >= 80 && !errors.Is(err, ErrMipmapSizeMismatch):
			t.Errorf("sample truncated to %d bytes: expected %v, got %v", n, ErrMipmapSizeMismatch, err)
		}
	}
}

func TestRead_SignatureMismatch(t *testing.T) {
	raw, _ := loadSample(t)
	data := append([]byte("VTX\x00"), raw[4:]...)

	if _, err := ReadFromStream(bytes.NewReader(data)); !errors.Is(err, ErrSignatureMismatch) {
		t.Errorf("expected %v, got %v", ErrSignatureMismatch, err)
	}
}

func TestRead_ReaderError(t *testing.T) {
	errRead := errors.New("read failed")

	if _, err := ReadFromStream(iotest.ErrReader(errRead)); !errors.Is(err, errRead) {
		t.Errorf("expected %v, got %v", errRead, err)
	}
	raw, _ := loadSample(t)
	stream := io.MultiReader(bytes.NewReader(raw[:1000]), iotest.ErrReader(errRead))
	if _, err := ReadFromStream(stream); !errors.Is(err, errRead) {
		t.Errorf("expected %v, got %v", errRead, err)
	}
}

func FuzzReadFromStream(f *testing.F) {
	raw, _ := loadSample(f)
	f.Add(raw)
	for _, minor := range []uint32{1, 2} {
		tt := newTestTexture()
		tt.minorVersion = minor
		tt.format = format.Dxt1
		tt.frames = 2
		data, _ := tt.build(f)
		f.Add(data)
	}
	cubemap := newTestTexture()
	cubemap.minorVersion = 4
	cubemap.flags = FlagEnvironmentMap
	cubemap.faces = 7
	cubemap.depth = 2
	cubemap.resources = []testResource{
		{tag: ResourceTagCRC, flags: ResourceFlagNoDataChunk, data: []byte{1, 2, 3, 4}},
		{tag: ResourceTagLowResImage},
		{tag: ResourceTagImage},
		{tag: ResourceTagKeyValues, data: []byte("key values")},
	}
	data, _ := cubemap.build(f)
	f.Add(data)

	f.Fuzz(func(t *testing.T, data []byte) {
		v, err := ReadFromStream(bytes.NewReader(data))
		if err != nil {
			return
		}
		// A decoded texture must be safe to use
		_ = v.Image()
		_ = v.LowResImageData()
		for level := 0; level < v.MipmapCount(); level++ {
			_, _, depth := v.MipmapDimensions(level)
			if _, err := v.Mipmap(level, v.FrameCount()-1, v.FaceCount()-1, depth-1); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func encodeHeader(t testing.TB, h Header) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := binaryWrite(buf, &h); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
