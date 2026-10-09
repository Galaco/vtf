package vtfimage_test

import (
	"fmt"
	"image"
	"log"
	"os"

	_ "github.com/galaco/vtf/v2/vtfimage"
)

func Example() {
	file, err := os.Open("../testdata/test.vtf")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	img, name, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name, img.Bounds())
	// Output: vtf (0,0)-(512,128)
}
