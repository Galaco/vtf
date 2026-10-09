[![Go Reference](https://pkg.go.dev/badge/github.com/galaco/vtf/v2.svg)](https://pkg.go.dev/github.com/galaco/vtf/v2)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/vtf)](https://goreportcard.com/report/github.com/galaco/vtf)
[![Go](https://github.com/Galaco/vtf/actions/workflows/go.yml/badge.svg)](https://github.com/Galaco/vtf/actions/workflows/go.yml)

# vtf
> 0 dependency parser for Valves own .vtf (Valve Texture Format) Source Engine textures.

```
go get github.com/galaco/vtf/v2
```

### Features
* Supports versions 7.0-7.5
* Full header data
* Low resolution thumbnail loading
* Complete mipmap + high-resolution texture loading, including animated frames, cubemaps and volume textures
* v7.3+ resources (key values, sprite sheets, CRCs etc.)
* Streaming reads: only the bytes the texture occupies are read, and sizes claimed by a header are never trusted for allocation
* Optional conversion to `image.Image`, with the `vtfimage` package

### Usage
```go
package main

import (
	"log"

	"github.com/galaco/vtf/v2"
)

func main() {
	texture, err := vtf.ReadFromFile("foo.vtf")
	if err != nil {
		log.Fatal(err)
	}

	header := texture.Header()
	log.Println(header.Width, header.Height, header.HighResImageFormat)

	// Raw data of the largest mipmap of the first frame, face and slice
	data := texture.Image()

	log.Println(len(data), "bytes")

	// Any mipmap (0 is the largest), frame, cubemap face or volume slice
	smallest := texture.MipmapCount() - 1
	mipmap, err := texture.Mipmap(smallest, 0, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	width, height, depth := texture.MipmapDimensions(smallest)
	log.Println(width, height, depth, len(mipmap))
}
```

Image data is returned in its stored format (`Header().HighResImageFormat`), and shares memory with the texture.

To read only the header:
```go
reader := vtf.NewReader(file)
header, err := reader.ReadHeader()
// Optionally, continue reading the rest of the texture
texture, err := reader.Read()
```

v7.3+ resources:
```go
if kv, ok := texture.Resource(vtf.ResourceTagKeyValues); ok {
	log.Println(string(kv.Data))
}
```

### Converting to image.Image
The `vtfimage` package converts image data to an `image.Image`, and registers the format with the `image` package:
```go
import (
	"image"
	"image/png"

	_ "github.com/galaco/vtf/v2/vtfimage"
)

img, _, err := image.Decode(file)
err = png.Encode(out, img)
```
Or convert a specific mipmap, or the thumbnail, with `vtfimage.Mipmap`, `vtfimage.LowResImage` and `vtfimage.Convert`.

Supported formats: all 8 bit RGB(A) orderings, 565/5551/4444 packed formats, I8, IA88, A8, blue screen formats,
DXT1/DXT3/DXT5, ATI1N/ATI2N, RGBA16161616 and floating point formats (clamped to [0, 1]).
P8, UV and depth buffer formats are not supported.

### Migrating from v1
* Import `github.com/galaco/vtf/v2`.
* `HighResImageData()`, `MipmapsForFrame()` and `HighestResolutionImageForFrame()` are replaced by
  `Mipmap(level, frame, face, slice)`, `MipmapCount()`, `FrameCount()`, `FaceCount()` and `MipmapDimensions()`.
  **Mipmap level 0 is now the largest**, where v1 indexed mipmaps smallest first.
* `Header.Flags` is `vtf.Flags`, and `Header.HighResImageFormat`/`LowResImageFormat` are `format.Format`.
  Use `uint32(...)` where a plain integer is needed, and `Flags.Has` to test flags.
* Errors are named `ErrX` rather than `ErrorX`, e.g. `ErrSignatureMismatch`, `ErrMipmapSizeMismatch`.
* `FlagSRGB` is now `0x40`; v1 incorrectly defined it as `0xFFFF`.

### Whats missing
* Modify/export functionality
* Depth buffer formats (`NV_DST16` etc.), which are only used for render targets

### Contributing
No where near all the possible texture configurations have been tested. It's possible some could cause issues. Any issues 
(including offending file) are greatly appreciated.
