package vtf

import (
	"bytes"
	"errors"
	"testing"

	"github.com/galaco/vtf/format"
)

func TestRead_Cubemap(t *testing.T) {
	tests := []struct {
		name         string
		minorVersion uint32
		storedFaces  int
		firstFrame   uint16
	}{
		{"7.0 cubemap", 0, 6, 0},
		{"7.1 cubemap with spheremap", 1, 7, 0},
		{"7.2 cubemap with spheremap", 2, 7, 0},
		{"7.2 cubemap", 2, 6, 0xFFFF},
		{"7.4 cubemap with spheremap", 4, 7, 0},
		{"7.4 cubemap", 4, 6, 0xFFFF},
		{"7.5 cubemap", 5, 6, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTestTexture()
			tt.minorVersion = tc.minorVersion
			tt.flags = FlagEnvironmentMap
			tt.firstFrame = tc.firstFrame
			tt.faces = tc.storedFaces
			tt.frames = 2
			tt.format = format.Dxt5
			data, images := tt.build(t)

			v, err := ReadFromStream(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if faces := len(v.HighResImageData()[0][0]); faces != tc.storedFaces {
				t.Errorf("got %d faces, want %d", faces, tc.storedFaces)
			}
			assertImages(t, v, images)
		})
	}
}

func TestRead_CubemapTruncated(t *testing.T) {
	tt := newTestTexture()
	tt.minorVersion = 5
	tt.flags = FlagEnvironmentMap
	tt.faces = 6
	data, _ := tt.build(t)

	if _, err := ReadFromStream(bytes.NewReader(data[:len(data)-1])); !errors.Is(err, ErrorMipmapSizeMismatch) {
		t.Errorf("expected %v, got %v", ErrorMipmapSizeMismatch, err)
	}
}

func TestRead_VolumeTexture(t *testing.T) {
	tt := newTestTexture()
	tt.width, tt.height, tt.depth = 16, 8, 4
	tt.frames = 2
	data, images := tt.build(t)

	v, err := ReadFromStream(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	assertImages(t, v, images)

	// Each mipmap halves depth too: 4, 2, 1, 1, 1 slices from largest to smallest
	mipmaps := v.HighResImageData()
	for level, want := range []int{4, 2, 1, 1, 1} {
		if got := len(mipmaps[len(mipmaps)-1-level][0][0]); got != want {
			t.Errorf("mipmap %d: got %d slices, want %d", level, got, want)
		}
	}
}

func TestRead_Resources(t *testing.T) {
	keyValues := []byte(`"Information" { "Author" "galaco" }`)
	sheet := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	crc := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	tests := []struct {
		name      string
		resources []testResource
		reverse   bool
	}{
		{name: "images only"},
		{
			name: "sheet between images",
			resources: []testResource{
				{tag: ResourceTagLowResImage},
				{tag: ResourceTagSheet, data: sheet},
				{tag: ResourceTagImage},
			},
		},
		{
			name: "key values after image",
			resources: []testResource{
				{tag: ResourceTagLowResImage},
				{tag: ResourceTagImage},
				{tag: ResourceTagKeyValues, data: keyValues},
			},
		},
		{
			name: "everything",
			resources: []testResource{
				{tag: ResourceTagCRC, flags: ResourceFlagNoDataChunk, data: crc},
				{tag: ResourceTagLowResImage},
				{tag: ResourceTagSheet, data: sheet},
				{tag: ResourceTagImage},
				{tag: ResourceTagKeyValues, data: keyValues},
			},
		},
		{
			name: "directory order differs from data order",
			resources: []testResource{
				{tag: ResourceTagLowResImage},
				{tag: ResourceTagImage},
				{tag: ResourceTagKeyValues, data: keyValues},
				{tag: ResourceTagSheet, data: sheet},
			},
			reverse: true,
		},
	}
	for _, minor := range []uint32{3, 4, 5} {
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tt := newTestTexture()
				tt.minorVersion = minor
				tt.format = format.Dxt1
				tt.resources = tc.resources
				tt.reverseDirectory = tc.reverse
				data, images := tt.build(t)

				v, err := ReadFromStream(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("7.%d: %v", minor, err)
				}
				assertImages(t, v, images)
				if !bytes.Equal(v.LowResImageData(), bytes.Repeat([]byte{testLowResFill}, 128)) {
					t.Errorf("7.%d: low resolution image does not match", minor)
				}

				want := tt.resourceList()
				if len(v.Resources()) != len(want) {
					t.Fatalf("7.%d: got %d resources, want %d", minor, len(v.Resources()), len(want))
				}
				for _, w := range want {
					got, ok := v.Resource(w.tag)
					if !ok {
						t.Errorf("7.%d: missing resource %q", minor, w.tag)
						continue
					}
					if got.Flags != w.flags || !bytes.Equal(got.Data, w.data) {
						t.Errorf("7.%d: resource %q: got flags %d data %q, want flags %d data %q",
							minor, w.tag, got.Flags, got.Data, w.flags, w.data)
					}
				}
			})
		}
	}
}

func TestRead_ResourcesWithoutLowResImage(t *testing.T) {
	tt := newTestTexture()
	tt.minorVersion = 4
	tt.resources = []testResource{{tag: ResourceTagImage}}
	data, images := tt.build(t)

	v, err := ReadFromStream(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(v.LowResImageData()) != 0 {
		t.Errorf("expected no low resolution image, got %d bytes", len(v.LowResImageData()))
	}
	assertImages(t, v, images)
}

func TestRead_ResourcesTrailingBytes(t *testing.T) {
	tt := newTestTexture()
	tt.minorVersion = 5
	tt.trailing = []byte("trailing")
	data, images := tt.build(t)

	v, err := ReadFromStream(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	assertImages(t, v, images)
}

func TestRead_InvalidResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []testResource
		corrupt   func([]byte)
		wantErr   error
	}{
		{
			name:      "missing image",
			resources: []testResource{{tag: ResourceTagLowResImage}},
			wantErr:   ErrorInvalidResource,
		},
		{
			name:      "image offset out of bounds",
			resources: []testResource{{tag: ResourceTagLowResImage}, {tag: ResourceTagImage}},
			corrupt:   func(data []byte) { putUint32(data[92:], 0xFFFFFFF0) },
			wantErr:   ErrorMipmapSizeMismatch,
		},
		{
			name:      "low res offset out of bounds",
			resources: []testResource{{tag: ResourceTagLowResImage}, {tag: ResourceTagImage}},
			corrupt:   func(data []byte) { putUint32(data[84:], uint32(len(data))-4) },
			wantErr:   ErrorMipmapSizeMismatch,
		},
		{
			name: "data chunk size out of bounds",
			resources: []testResource{
				{tag: ResourceTagLowResImage},
				{tag: ResourceTagImage},
				{tag: ResourceTagKeyValues, data: []byte("kv")},
			},
			corrupt: func(data []byte) { putUint32(data[len(data)-6:], 1000) },
			wantErr: ErrorMipmapSizeMismatch,
		},
		{
			name: "data chunk image overlap",
			resources: []testResource{
				{tag: ResourceTagLowResImage},
				{tag: ResourceTagKeyValues, data: []byte("kv")},
				{tag: ResourceTagImage},
			},
			// Point the key values into the middle of the image, so the image is cut short
			corrupt: func(data []byte) { putUint32(data[92:], uint32(len(data))-16) },
			wantErr: ErrorMipmapSizeMismatch,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newTestTexture()
			tt.minorVersion = 5
			tt.resources = tc.resources
			data, _ := tt.build(t)
			if tc.corrupt != nil {
				tc.corrupt(data)
			}

			if _, err := ReadFromStream(bytes.NewReader(data)); !errors.Is(err, tc.wantErr) {
				t.Errorf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestRead_ResourcesTruncated(t *testing.T) {
	tt := newTestTexture()
	tt.minorVersion = 5
	tt.format = format.Dxt1
	tt.resources = []testResource{
		{tag: ResourceTagCRC, flags: ResourceFlagNoDataChunk, data: []byte{1, 2, 3, 4}},
		{tag: ResourceTagLowResImage},
		{tag: ResourceTagImage},
		{tag: ResourceTagKeyValues, data: []byte("key values")},
	}
	data, _ := tt.build(t)

	for n := 0; n < len(data); n++ {
		if _, err := ReadFromStream(bytes.NewReader(data[:n])); err == nil {
			t.Fatalf("truncated to %d of %d bytes: expected error", n, len(data))
		}
	}
}

func putUint32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}
