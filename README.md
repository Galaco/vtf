[![GoDoc](https://godoc.org/github.com/Galaco/vtf?status.svg)](https://godoc.org/github.com/Galaco/vtf)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/vtf)](https://goreportcard.com/badge/github.com/galaco/vtf)
[![codecov](https://codecov.io/gh/Galaco/vtf/branch/master/graph/badge.svg)](https://codecov.io/gh/Galaco/vtf)
[![CircleCI](https://circleci.com/gh/Galaco/vtf.svg?style=svg)](https://circleci.com/gh/Galaco/vtf)


# vtf
> 0 dependency parser for Valves own .vtf (Valve Texture Format) Source Engine textures.

### Features
* Supports versions 7.0-7.5
* Full header data
* Low resolution thumbnail loading
* Complete mipmap + high-resolution texture loading, including animated frames, cubemaps and volume textures
* v7.3+ resources (key values, sprite sheets, CRCs etc.)
* Streaming reads: only the bytes the texture occupies are read, and claimed sizes are never trusted for allocation

### Usage
```go
package main

import (
	"log"

	"github.com/galaco/vtf"
)

func main() {
	texture, err := vtf.ReadFromFile("foo.vtf")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(texture.Header().Width)
}
```

To read only the header:
```go
header, err := vtf.NewReader(file).ReadHeader()
```

Image data is returned in its stored format; see `Header().HighResImageFormat` and the `format` package.
`HighResImageData()` is indexed `[mipmap][frame][face][slice]`, with mipmaps ordered smallest first.

### Whats missing
* Modify/export functionality
* Depth buffer formats (`NV_DST16` etc.), which are only used for render targets

### What won't this ever do?
* Colour format transformation. Header properties `LowResImageFormat` and `HighResImageFormat` will provide the format.

### Contributing
No where near all the possible texture configurations have been tested. It's possible some could cause issues. Any issues 
(including offending file) are greatly appreciated.
