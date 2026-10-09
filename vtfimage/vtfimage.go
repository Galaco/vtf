// Package vtfimage converts vtf image data to image.Image.
//
// Importing it registers the "vtf" format with the image package, so
// image.Decode and image.DecodeConfig can read textures:
//
//	import _ "github.com/galaco/vtf/v2/vtfimage"
package vtfimage

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/galaco/vtf/v2"
	"github.com/galaco/vtf/v2/format"
	"github.com/galaco/vtf/v2/internal"
)

var (
	// ErrUnsupportedFormat occurs when converting a format that has no image.Image equivalent
	ErrUnsupportedFormat = errors.New("vtfimage: unsupported format")
	// ErrShortData occurs when image data is smaller than its dimensions and format require
	ErrShortData = errors.New("vtfimage: image data too short")
)

func init() {
	image.RegisterFormat("vtf", "VTF\x00", Decode, DecodeConfig)
}

// Decode reads a texture, returning the largest mipmap of its first frame, face and slice
func Decode(r io.Reader) (image.Image, error) {
	texture, err := vtf.ReadFromStream(r)
	if err != nil {
		return nil, err
	}

	header := texture.Header()
	return Convert(texture.Image(), int(header.Width), int(header.Height), header.HighResImageFormat)
}

// DecodeConfig returns the colour model and dimensions of a texture's largest mipmap,
// reading only its header
func DecodeConfig(r io.Reader) (image.Config, error) {
	header, err := vtf.NewReader(r).ReadHeader()
	if err != nil {
		return image.Config{}, err
	}
	model, err := ColorModel(header.HighResImageFormat)
	if err != nil {
		return image.Config{}, err
	}

	return image.Config{
		ColorModel: model,
		Width:      int(header.Width),
		Height:     int(header.Height),
	}, nil
}

// Mipmap converts a single image of a texture. Mipmap level 0 is the largest.
func Mipmap(texture *vtf.Vtf, level, frame, face, slice int) (image.Image, error) {
	data, err := texture.Mipmap(level, frame, face, slice)
	if err != nil {
		return nil, err
	}

	width, height, _ := texture.MipmapDimensions(level)
	return Convert(data, width, height, texture.Header().HighResImageFormat)
}

// LowResImage converts a texture's low resolution thumbnail. It returns nil if the texture has none.
func LowResImage(texture *vtf.Vtf) (image.Image, error) {
	data := texture.LowResImageData()
	if len(data) == 0 {
		return nil, nil
	}

	header := texture.Header()
	return Convert(data, int(header.LowResImageWidth), int(header.LowResImageHeight), header.LowResImageFormat)
}

// ColorModel returns the colour model that Convert produces for a format
func ColorModel(f format.Format) (color.Model, error) {
	switch f {
	case format.I8, format.Ati1n:
		return color.GrayModel, nil
	case format.A8:
		return color.AlphaModel, nil
	case format.R32F:
		return color.Gray16Model, nil
	case format.RGBA16161616, format.RGBA16161616F, format.RGB323232F, format.RGBA32323232F:
		return color.NRGBA64Model, nil
	case format.RGBA8888, format.ABGR8888, format.RGB888, format.BGR888, format.RGB565, format.BGR565,
		format.IA88, format.RGB888BLUESCREEN, format.BGR888BLUESCREEN, format.ARGB8888, format.BGRA8888,
		format.BGRX8888, format.BGRX5551, format.BGRA4444, format.BGRA5551,
		format.Dxt1, format.Dxt1OneBitAlpha, format.Dxt3, format.Dxt5, format.Ati2n:
		return color.NRGBAModel, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, f)
}

// Convert converts width x height image data stored in format f. The returned image
// does not share memory with data.
//
// Floating point formats are clamped to [0, 1]. DXT1 is treated as opaque, while
// DXT1_ONEBITALPHA decodes transparent pixels. ATI2N channels are decoded into red and
// green, without reconstructing a normal's blue channel. Paletted, UV and depth
// formats are not supported.
func Convert(data []byte, width, height int, f format.Format) (image.Image, error) {
	if _, err := ColorModel(f); err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("vtfimage: invalid dimensions %dx%d", width, height)
	}
	size, err := internal.ComputeSizeOfMipmapData(width, height, f)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, f)
	}
	if len(data) < size {
		return nil, fmt.Errorf("%w: %s %dx%d needs %d bytes, have %d", ErrShortData, f, width, height, size, len(data))
	}

	rect := image.Rect(0, 0, width, height)
	switch f {
	case format.I8:
		img := image.NewGray(rect)
		copy(img.Pix, data)
		return img, nil
	case format.A8:
		img := image.NewAlpha(rect)
		copy(img.Pix, data)
		return img, nil
	case format.Dxt1, format.Dxt1OneBitAlpha, format.Dxt3, format.Dxt5, format.Ati2n:
		return decodeBlocks(data, rect, f), nil
	case format.Ati1n:
		return decodeBC4(data, rect), nil
	case format.R32F, format.RGBA16161616, format.RGBA16161616F, format.RGB323232F, format.RGBA32323232F:
		return convertWide(data, rect, f), nil
	}

	return convertPacked(data, rect, f), nil
}
