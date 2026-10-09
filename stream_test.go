package vtf

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"testing"
	"testing/iotest"

	"github.com/galaco/vtf/v2/format"
)

// countingReader counts the bytes read from it
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

func resourceTexture() testTexture {
	tt := newTestTexture()
	tt.minorVersion = 5
	tt.format = format.Dxt5
	tt.frames = 2
	tt.resources = []testResource{
		{tag: ResourceTagCRC, flags: ResourceFlagNoDataChunk, data: []byte{1, 2, 3, 4}},
		{tag: ResourceTagLowResImage},
		{tag: ResourceTagSheet, data: []byte("sheet")},
		{tag: ResourceTagImage},
		{tag: ResourceTagKeyValues, data: []byte("key values")},
	}
	return tt
}

func TestReader_ReadHeader(t *testing.T) {
	raw, want := loadSample(t)
	resources, _ := resourceTexture().build(t)

	for name, data := range map[string][]byte{"7.2": raw, "7.5 with resources": resources} {
		t.Run(name, func(t *testing.T) {
			stream := &countingReader{r: bytes.NewReader(data)}
			header, err := NewReader(stream).ReadHeader()
			if err != nil {
				t.Fatal(err)
			}
			// Only the header is read, including the resource directory
			if stream.n != int(header.HeaderSize) {
				t.Errorf("read %d bytes, want header size %d", stream.n, header.HeaderSize)
			}
		})
	}

	header, err := NewReader(bytes.NewReader(raw)).ReadHeader()
	if err != nil {
		t.Fatal(err)
	}
	if *header != want.Header() {
		t.Errorf("ReadHeader: got %+v, want %+v", *header, want.Header())
	}
}

func TestReader_ReadHeaderThenRead(t *testing.T) {
	data, images := resourceTexture().build(t)

	reader := NewReader(bytes.NewReader(data))
	header, err := reader.ReadHeader()
	if err != nil {
		t.Fatal(err)
	}
	// The returned header is a copy
	header.Width = 1

	v, err := reader.Read()
	if err != nil {
		t.Fatal(err)
	}
	if v.Header().Width != 16 {
		t.Errorf("width: got %d, want 16", v.Header().Width)
	}
	assertImages(t, v, images)

	again, err := reader.Read()
	if again != v || err != nil {
		t.Errorf("second Read: got %p, %v, want %p", again, err, v)
	}
}

func TestReader_InvalidHeader(t *testing.T) {
	tt := newTestTexture()
	tt.frames = 0
	data, _ := tt.build(t)

	reader := NewReader(bytes.NewReader(data))
	if _, err := reader.ReadHeader(); !errors.Is(err, ErrInvalidDimensions) {
		t.Errorf("ReadHeader: expected %v, got %v", ErrInvalidDimensions, err)
	}
	if _, err := reader.Read(); !errors.Is(err, ErrInvalidDimensions) {
		t.Errorf("Read: expected %v, got %v", ErrInvalidDimensions, err)
	}
}

func TestReader_ReadHeaderAfterFailedRead(t *testing.T) {
	data, _ := resourceTexture().build(t)

	reader := NewReader(bytes.NewReader(data[:len(data)-1]))
	if _, err := reader.Read(); !errors.Is(err, ErrMipmapSizeMismatch) {
		t.Fatalf("expected %v, got %v", ErrMipmapSizeMismatch, err)
	}
	if header, err := reader.ReadHeader(); err != nil || header.Width != 16 {
		t.Errorf("ReadHeader after failed Read: got %v, %v", header, err)
	}
}

func TestReader_ConsecutiveTextures(t *testing.T) {
	raw, _ := loadSample(t)
	resources, resourceImages := resourceTexture().build(t)
	legacy := newTestTexture()
	legacy.minorVersion = 1
	legacy.frames = 3
	legacyData, legacyImages := legacy.build(t)

	// Each texture is read up to its end, leaving the next one in the stream
	stream := io.MultiReader(bytes.NewReader(resources), bytes.NewReader(legacyData), bytes.NewReader(raw))
	first, err := ReadFromStream(stream)
	if err != nil {
		t.Fatal(err)
	}
	assertImages(t, first, resourceImages)
	second, err := ReadFromStream(stream)
	if err != nil {
		t.Fatal(err)
	}
	assertImages(t, second, legacyImages)
	third, err := ReadFromStream(stream)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(third.Image(), raw[112:]) {
		t.Error("third texture does not match")
	}
}

func TestReader_ShortReads(t *testing.T) {
	raw, sample := loadSample(t)
	resources, resourceImages := resourceTexture().build(t)
	readers := map[string]func(io.Reader) io.Reader{
		"one byte": iotest.OneByteReader,
		"half":     iotest.HalfReader,
		"data err": iotest.DataErrReader,
	}
	for name, wrap := range readers {
		t.Run(name, func(t *testing.T) {
			v, err := ReadFromStream(wrap(bytes.NewReader(raw)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(v.Image(), sample.Image()) {
				t.Error("sample image does not match")
			}

			v, err = ReadFromStream(wrap(bytes.NewReader(resources)))
			if err != nil {
				t.Fatal(err)
			}
			assertImages(t, v, resourceImages)
		})
	}
}

func TestReader_StreamError(t *testing.T) {
	raw, _ := loadSample(t)

	_, err := ReadFromStream(iotest.TimeoutReader(bytes.NewReader(raw)))
	if !errors.Is(err, iotest.ErrTimeout) {
		t.Errorf("expected %v, got %v", iotest.ErrTimeout, err)
	}
}

func TestReader_ClaimedSizeIsNotPreallocated(t *testing.T) {
	// A small file claiming to hold ~20GB of image data
	tt := newTestTexture()
	tt.width, tt.height = 16384, 16384
	tt.format = format.RGBA32323232F
	tt.mipmaps = 1
	tt.frames = 1024
	tt.lowResFormat = format.None
	h := tt.header()
	data := append(encodeHeader(t, h), make([]byte, int(h.HeaderSize)-headerStructSize+4096)...)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := ReadFromStream(bytes.NewReader(data))
	runtime.ReadMemStats(&after)

	if !errors.Is(err, ErrMipmapSizeMismatch) {
		t.Errorf("expected %v, got %v", ErrMipmapSizeMismatch, err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 4*maxPreallocSize {
		t.Errorf("allocated %d bytes reading a %d byte file", allocated, len(data))
	}
}

func TestFlags(t *testing.T) {
	// Values from VTFLib and the Source SDK
	flags := map[string][2]Flags{
		"PointSampling":         {FlagPointSampling, 0x1},
		"SRGB":                  {FlagSRGB, 0x40},
		"NormalMap":             {FlagNormalMap, 0x80},
		"NoMinimumMipmap":       {FlagNoMinimumMipmap, 0x400},
		"EnvironmentMap":        {FlagEnvironmentMap, 0x4000},
		"PreSRGB":               {FlagPreSRGB, 0x80000},
		"NoDepthBuffer":         {FlagNoDepthBuffer, 0x800000},
		"ClampU":                {FlagClampU, 0x2000000},
		"SSBump":                {FlagSSBump, 0x8000000},
		"UnfilterableOK":        {FlagUnfilterableOK, 0x10000000},
		"Border":                {FlagBorder, 0x20000000},
		"SpecVarRed":            {FlagSpecVarRed, 0x40000000},
		"SpecVarAlpha":          {FlagSpecVarAlpha, 0x80000000},
		"PreMultiplyColorByMip": {FlagPreMultiplyColorByOneOverMipmapLevel, 0x100000},
	}
	for name, values := range flags {
		if values[0] != values[1] {
			t.Errorf("Flag%s: got %#x, want %#x", name, values[0], values[1])
		}
	}

	set := FlagNormalMap | FlagNoMipmaps
	if !set.Has(FlagNormalMap) || !set.Has(FlagNormalMap|FlagNoMipmaps) || set.Has(FlagNormalMap|FlagSRGB) {
		t.Error("Flags.Has does not match set flags")
	}
}

func BenchmarkReadFromStream(b *testing.B) {
	raw, _ := loadSample(b)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ReadFromStream(bytes.NewReader(raw)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadHeader(b *testing.B) {
	raw, _ := loadSample(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := NewReader(bytes.NewReader(raw)).ReadHeader(); err != nil {
			b.Fatal(err)
		}
	}
}
