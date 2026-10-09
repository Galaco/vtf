package vtfimage

import (
	"encoding/binary"
	"image"

	"github.com/galaco/vtf/v2/format"
)

// decodeBlocks decodes BC1-3 and BC5 (DXT1, DXT3, DXT5 and ATI2N) compressed data
func decodeBlocks(data []byte, rect image.Rectangle, f format.Format) image.Image {
	img := image.NewNRGBA(rect)
	blockSize := 16
	if f == format.Dxt1 || f == format.Dxt1OneBitAlpha {
		blockSize = 8
	}

	var pixels [16][4]uint8
	offset := 0
	for by := 0; by < rect.Dy(); by += 4 {
		for bx := 0; bx < rect.Dx(); bx += 4 {
			block := data[offset : offset+blockSize]
			offset += blockSize

			switch f {
			case format.Dxt1:
				decodeColor(block, &pixels, true, false)
			case format.Dxt1OneBitAlpha:
				decodeColor(block, &pixels, true, true)
			case format.Dxt3:
				decodeColor(block[8:], &pixels, false, false)
				alpha := binary.LittleEndian.Uint64(block)
				for i := range pixels {
					pixels[i][3] = uint8(alpha>>(4*i)&0xF) * 17
				}
			case format.Dxt5:
				decodeColor(block[8:], &pixels, false, false)
				decodeChannel(block, &pixels, 3)
			case format.Ati2n:
				decodeChannel(block, &pixels, 0)
				decodeChannel(block[8:], &pixels, 1)
				for i := range pixels {
					pixels[i][2], pixels[i][3] = 0, 255
				}
			}
			writeBlock(img.Pix, img.Stride, 4, rect, bx, by, &pixels)
		}
	}
	return img
}

// decodeBC4 decodes BC4 (ATI1N) compressed data
func decodeBC4(data []byte, rect image.Rectangle) image.Image {
	img := image.NewGray(rect)

	var pixels [16][4]uint8
	offset := 0
	for by := 0; by < rect.Dy(); by += 4 {
		for bx := 0; bx < rect.Dx(); bx += 4 {
			decodeChannel(data[offset:offset+8], &pixels, 0)
			offset += 8
			writeBlock(img.Pix, img.Stride, 1, rect, bx, by, &pixels)
		}
	}
	return img
}

// writeBlock copies the first channels of a decoded 4x4 block into an image, clipped to its bounds
func writeBlock(pix []uint8, stride, channels int, rect image.Rectangle, bx, by int, pixels *[16][4]uint8) {
	for y := 0; y < 4 && by+y < rect.Dy(); y++ {
		for x := 0; x < 4 && bx+x < rect.Dx(); x++ {
			copy(pix[(by+y)*stride+(bx+x)*channels:], pixels[y*4+x][:channels])
		}
	}
}

// decodeColor decodes a BC1 colour block. allowThreeColor enables the 3 colour mode used when
// the first endpoint is not greater than the second; transparent makes its 4th colour transparent.
func decodeColor(block []byte, pixels *[16][4]uint8, allowThreeColor, transparent bool) {
	c0 := binary.LittleEndian.Uint16(block[0:])
	c1 := binary.LittleEndian.Uint16(block[2:])

	var palette [4][4]uint8
	palette[0] = rgb565(c0)
	palette[1] = rgb565(c1)
	if c0 > c1 || !allowThreeColor {
		for c := 0; c < 3; c++ {
			palette[2][c] = uint8((2*uint(palette[0][c]) + uint(palette[1][c])) / 3)
			palette[3][c] = uint8((uint(palette[0][c]) + 2*uint(palette[1][c])) / 3)
		}
		palette[2][3], palette[3][3] = 255, 255
	} else {
		for c := 0; c < 3; c++ {
			palette[2][c] = uint8((uint(palette[0][c]) + uint(palette[1][c])) / 2)
		}
		palette[2][3], palette[3][3] = 255, 255
		if transparent {
			palette[3][3] = 0
		}
	}

	indices := binary.LittleEndian.Uint32(block[4:])
	for i := range pixels {
		pixels[i] = palette[indices>>(2*i)&3]
	}
}

// rgb565 expands a BC1 endpoint, which stores red in the most significant bits
func rgb565(c uint16) [4]uint8 {
	r, g, b := uint32(c>>11), uint32(c>>5&0x3F), uint32(c&0x1F)
	return [4]uint8{uint8(r<<3 | r>>2), uint8(g<<2 | g>>4), uint8(b<<3 | b>>2), 255}
}

// decodeChannel decodes a BC3 alpha / BC4 block into one channel of pixels
func decodeChannel(block []byte, pixels *[16][4]uint8, channel int) {
	var values [8]uint8
	values[0], values[1] = block[0], block[1]
	v0, v1 := uint(block[0]), uint(block[1])
	if v0 > v1 {
		for i := uint(1); i < 7; i++ {
			values[i+1] = uint8(((7-i)*v0 + i*v1) / 7)
		}
	} else {
		for i := uint(1); i < 5; i++ {
			values[i+1] = uint8(((5-i)*v0 + i*v1) / 5)
		}
		values[6], values[7] = 0, 255
	}

	// 3 bit indices, packed into 48 bits
	indices := uint64(binary.LittleEndian.Uint16(block[2:])) |
		uint64(binary.LittleEndian.Uint32(block[4:]))<<16
	for i := range pixels {
		pixels[i][channel] = values[indices>>(3*i)&7]
	}
}
