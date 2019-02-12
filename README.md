[![GoDoc](https://godoc.org/github.com/Galaco/vtf?status.svg)](https://godoc.org/github.com/Galaco/vtf)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/vtf)](https://goreportcard.com/badge/github.com/galaco/vtf)
[![Build Status](https://travis-ci.com/Galaco/vtf.svg?branch=master)](https://travis-ci.com/Galaco/vtf)


# vtf
Parse Valves own .vtf format (literally Valve Texture Format) for Source Engine texture.

### Features
* Supports versions 7.1-7.5
* Full header data
* Low resolution thumbnail loading
* Complete mipmap + high-resolution texture loading

### Usage
```
import (  
	"github.com/galaco/vtf"
	"log"
	"os"
)

func main() {
  file,_ := os.LoadFile("foo.vtf")
  texture,err := vtf.ReadFromStream(file)
	if err != nil {
		log.Println(err)
	} else {
    log.Println(texture.GetHeader().Width)
  }
}

```

### Whats missing
* Resource data is ignored (besides mipmaps) in 7.3+
* Texture with depth > 1 are unsupported. This is very rare
* Textures with zslices > 1 are unsupported. This is very rare
* Modify/export functionality

### What won't this ever do?
* Colour format transformation. Header properties `LowResImageFormat` and `HighResImageFormat` will provide the format.
* (Probably) support depths or zslices > 1

### Contributing
No where near all the possible texture configurations have been tested. It's possible some could cause issues. Any issues (including offending file) are greatly appreciated.
