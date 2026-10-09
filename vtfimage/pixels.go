package vtfimage

import (
	"encoding/binary"
	"image"
	"math"

	"github.com/galaco/vtf/v2/format"
)

// byteOrder gives the byte offsets of each channel in a pixel, or -1 if absent
type byteOrder struct {
	size       int
	r, g, b, a int
}

var byteOrders = map[format.Format]byteOrder{
	format.RGBA8888: {4, 0, 1, 2, 3},
	format.ABGR8888: {4, 3, 2, 1, 0},
	format.RGB888:   {3, 0, 1, 2, -1},
	format.BGR888:   {3, 2, 1, 0, -1},
	// VTFLib decodes ARGB8888 as GBAR; this follows the format name, as sourcepp does
	format.ARGB8888:         {4, 1, 2, 3, 0},
	format.BGRA8888:         {4, 2, 1, 0, 3},
	format.BGRX8888:         {4, 2, 1, 0, -1},
	format.RGB888BLUESCREEN: {3, 0, 1, 2, -1},
	format.BGR888BLUESCREEN: {3, 2, 1, 0, -1},
	format.IA88:             {2, 0, 0, 0, 1},
}

// bitLayout gives the bit width of each channel of a 16 bit pixel, starting from the least significant bits
type bitLayout struct {
	order [4]int // channel index (0 r, 1 g, 2 b, 3 a, -1 unused) from least significant bits
	bits  [4]int
}

var bitLayouts = map[format.Format]bitLayout{
	format.RGB565:   {[4]int{0, 1, 2, -1}, [4]int{5, 6, 5, 0}},
	format.BGR565:   {[4]int{2, 1, 0, -1}, [4]int{5, 6, 5, 0}},
	format.BGRX5551: {[4]int{2, 1, 0, -1}, [4]int{5, 5, 5, 1}},
	format.BGRA5551: {[4]int{2, 1, 0, 3}, [4]int{5, 5, 5, 1}},
	format.BGRA4444: {[4]int{2, 1, 0, 3}, [4]int{4, 4, 4, 4}},
}

// convertPacked converts formats with 8 bit channels or 16 bit packed pixels
func convertPacked(data []byte, rect image.Rectangle, f format.Format) image.Image {
	img := image.NewNRGBA(rect)
	pixels := rect.Dx() * rect.Dy()

	if layout, ok := bitLayouts[f]; ok {
		for i := 0; i < pixels; i++ {
			pixel := uint32(binary.LittleEndian.Uint16(data[i*2:]))
			rgba := [4]uint8{0, 0, 0, 255}
			shift := 0
			for c, channel := range layout.order {
				bits := layout.bits[c]
				if channel >= 0 {
					rgba[channel] = expand(pixel>>shift&(1<<bits-1), bits)
				}
				shift += bits
			}
			copy(img.Pix[i*4:], rgba[:])
		}
		return img
	}

	order := byteOrders[f]
	blueScreen := f == format.RGB888BLUESCREEN || f == format.BGR888BLUESCREEN
	for i := 0; i < pixels; i++ {
		src := data[i*order.size:]
		dst := img.Pix[i*4 : i*4+4]
		dst[0], dst[1], dst[2], dst[3] = src[order.r], src[order.g], src[order.b], 255
		if order.a >= 0 {
			dst[3] = src[order.a]
		}
		// Pure blue is transparent
		if blueScreen && dst[0] == 0 && dst[1] == 0 && dst[2] == 255 {
			dst[3] = 0
		}
	}
	return img
}

// expand scales an n bit channel value to 8 bits, rounding to nearest
func expand(value uint32, bits int) uint8 {
	max := uint32(1)<<bits - 1
	return uint8((value*255 + max/2) / max)
}

// convertWide converts formats with 16 or 32 bit channels
func convertWide(data []byte, rect image.Rectangle, f format.Format) image.Image {
	pixels := rect.Dx() * rect.Dy()

	if f == format.R32F {
		img := image.NewGray16(rect)
		for i := 0; i < pixels; i++ {
			binary.BigEndian.PutUint16(img.Pix[i*2:], unitToUint16(math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))))
		}
		return img
	}

	channels, channelSize := 4, 2
	switch f {
	case format.RGB323232F:
		channels, channelSize = 3, 4
	case format.RGBA32323232F:
		channelSize = 4
	}

	img := image.NewNRGBA64(rect)
	for i := 0; i < pixels; i++ {
		dst := img.Pix[i*8 : i*8+8]
		binary.BigEndian.PutUint16(dst[6:], 0xFFFF)
		for c := 0; c < channels; c++ {
			src := data[(i*channels+c)*channelSize:]
			var value uint16
			switch f {
			case format.RGBA16161616:
				value = binary.LittleEndian.Uint16(src)
			case format.RGBA16161616F:
				value = unitToUint16(halfToFloat32(binary.LittleEndian.Uint16(src)))
			default:
				value = unitToUint16(math.Float32frombits(binary.LittleEndian.Uint32(src)))
			}
			binary.BigEndian.PutUint16(dst[c*2:], value)
		}
	}
	return img
}

// unitToUint16 clamps v to [0, 1] and scales it to 16 bits
func unitToUint16(v float32) uint16 {
	if !(v > 0) { // Includes NaN
		return 0
	}
	if v >= 1 {
		return 0xFFFF
	}
	return uint16(v*0xFFFF + 0.5)
}

// halfToFloat32 converts an IEEE 754 half precision float
func halfToFloat32(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exponent := uint32(h>>10) & 0x1F
	mantissa := uint32(h) & 0x3FF

	switch exponent {
	case 0:
		// Zero or subnormal
		return math.Float32frombits(sign) + float32(mantissa)*(1.0/(1<<24))*signum(sign)
	case 0x1F:
		// Infinity or NaN
		return math.Float32frombits(sign | 0x7F800000 | mantissa<<13)
	}
	return math.Float32frombits(sign | (exponent+112)<<23 | mantissa<<13)
}

func signum(sign uint32) float32 {
	if sign != 0 {
		return -1
	}
	return 1
}
