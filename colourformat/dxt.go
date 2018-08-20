package colourformat

import (
	"github.com/galaco/dxt"
	"image"
	"os"
	"image/png"
)

func FromDXT1(width int, height int, imagedata []byte) ([]uint8,error) {

	r := image.Rect(0, 0, width, height)
	img := dxt.NewDxt1(r)

	err := img.Decompress(imagedata)
	
	if err != nil {
		return nil,err
	}

	return img.Pix, nil
}

func FromDXT5(width int, height int, imagedata []byte) ([]uint8,error) {

	r := image.Rect(0, 0, width, height)
	img := dxt.NewDxt5(r)

	// Data shouldnt have header
	err := img.Decompress(imagedata, false)

	if err != nil {
		return nil,err
	}

	out, _ := os.Create("out" + string(len(imagedata)) + ".png")
	err = png.Encode(out, img)

	return img.Pix, nil
}
