# vtf
Parse Valves own .vtf format (literally Valve Texture Format) for Source Engine texture.

### Features
* Supports versions 7.1-7.5
* Full header data
* Low resolution thumbnail loading
* Complete mipmap loading

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
