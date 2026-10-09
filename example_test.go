package vtf_test

import (
	"fmt"
	"log"
	"os"

	"github.com/galaco/vtf/v2"
)

func ExampleReadFromFile() {
	texture, err := vtf.ReadFromFile("testdata/test.vtf")
	if err != nil {
		log.Fatal(err)
	}

	header := texture.Header()
	fmt.Printf("%dx%d, %d bytes\n", header.Width, header.Height, len(texture.Image()))
	// Output: 512x128, 196608 bytes
}

func ExampleNewReader() {
	file, err := os.Open("testdata/test.vtf")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read just the header, then decide whether to read the image data
	reader := vtf.NewReader(file)
	header, err := reader.ReadHeader()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("v%d.%d %dx%d\n", header.Version[0], header.Version[1], header.Width, header.Height)

	if !header.Flags.Has(vtf.FlagEnvironmentMap) {
		texture, err := reader.Read()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(len(texture.Image()), "bytes")
	}
	// Output:
	// v7.2 512x128
	// 196608 bytes
}

func ExampleVtf_Mipmap() {
	texture, err := vtf.ReadFromFile("testdata/test.vtf")
	if err != nil {
		log.Fatal(err)
	}

	for level := 0; level < texture.MipmapCount(); level++ {
		data, err := texture.Mipmap(level, 0, 0, 0)
		if err != nil {
			log.Fatal(err)
		}
		width, height, _ := texture.MipmapDimensions(level)
		fmt.Printf("mipmap %d: %dx%d %s, %d bytes\n", level, width, height, texture.Header().HighResImageFormat, len(data))
	}
	// Output: mipmap 0: 512x128 BGR888, 196608 bytes
}
