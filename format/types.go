package format

import "strconv"

// Format is a pixel format that colour data is stored in
type Format uint32

const (
	// RGBA8888 RGBA (4bytes)
	RGBA8888 = Format(0)
	// ABGR8888 ABGR (4bytes)
	ABGR8888 = Format(1)
	// RGB888 RGB (3bytes)
	RGB888 = Format(2)
	// BGR888 BGR (3bytes)
	BGR888 = Format(3)
	// RGB565 RGB (2bytes)
	RGB565 = Format(4)
	// I8 luminance (1byte)
	I8 = Format(5)
	// IA88 luminance + alpha (2bytes)
	IA88 = Format(6)
	// P8 paletted (1byte). Unused by Source
	P8 = Format(7)
	// A8 alpha (1byte)
	A8 = Format(8)
	// RGB888BLUESCREEN RGB (3bytes)
	RGB888BLUESCREEN = Format(9)
	// BGR888BLUESCREEN RBGR (3bytes)
	BGR888BLUESCREEN = Format(10)
	// ARGB8888 ARGB (4bytes)
	ARGB8888 = Format(11)
	// BGRA8888 BGRA (4bytes)
	BGRA8888 = Format(12)
	// Dxt1 BC1 compressed (8bytes per 4x4 block)
	Dxt1 = Format(13)
	// Dxt3 BC2 compressed (16bytes per 4x4 block)
	Dxt3 = Format(14)
	// Dxt5 BC3 compressed (16bytes per 4x4 block)
	Dxt5 = Format(15)
	// BGRX8888 BGRX - X may not be alpha (4bytes)
	BGRX8888 = Format(16)
	// BGR565 BGR (2bytes)
	BGR565 = Format(17)
	// BGRX5551 BGRX - X may not be alpha (2bytes)
	BGRX5551 = Format(18)
	// BGRA4444 BGRA (2bytes)
	BGRA4444 = Format(19)
	// Dxt1OneBitAlpha BC1 compressed with 1bit alpha (8bytes per 4x4 block)
	Dxt1OneBitAlpha = Format(20)
	// BGRA5551 BGRA opacity either 0% or 100% (2bytes)
	BGRA5551 = Format(21)
	// UV88 UV (2bytes)
	UV88 = Format(22)
	// UVWQ8888 UVWQ (4bytes)
	UVWQ8888 = Format(23)
	// RGBA16161616F RGBA (8bytes) as half floats
	RGBA16161616F = Format(24)
	// RGBA16161616 RGBA (8bytes)
	RGBA16161616 = Format(25)
	// UVLX8888 UVLX (4bytes)
	UVLX8888 = Format(26)
	// R32F R (4bytes) as float
	R32F = Format(27)
	// RGB323232F RGB (12bytes) as floats
	RGB323232F = Format(28)
	// RGBA32323232F RGBA (16bytes) as floats
	RGBA32323232F = Format(29)
	// NvDst16 nVidia depth buffer format. Render targets only
	NvDst16 = Format(30)
	// NvDst24 nVidia depth buffer format. Render targets only
	NvDst24 = Format(31)
	// NvIntz nVidia depth buffer format. Render targets only
	NvIntz = Format(32)
	// NvRawz nVidia depth buffer format. Render targets only
	NvRawz = Format(33)
	// AtiDst16 ATI depth buffer format. Render targets only
	AtiDst16 = Format(34)
	// AtiDst24 ATI depth buffer format. Render targets only
	AtiDst24 = Format(35)
	// NvNull nVidia null render target format. Render targets only
	NvNull = Format(36)
	// Ati2n BC5/3Dc two channel compressed (16bytes per 4x4 block)
	Ati2n = Format(37)
	// Ati1n BC4 single channel compressed (8bytes per 4x4 block)
	Ati1n = Format(38)

	// None indicates no image is stored, e.g. a texture without a low resolution thumbnail
	None = Format(0xFFFFFFFF)
)

var names = [...]string{
	"RGBA8888", "ABGR8888", "RGB888", "BGR888", "RGB565", "I8", "IA88", "P8", "A8",
	"RGB888_BLUESCREEN", "BGR888_BLUESCREEN", "ARGB8888", "BGRA8888", "DXT1", "DXT3", "DXT5",
	"BGRX8888", "BGR565", "BGRX5551", "BGRA4444", "DXT1_ONEBITALPHA", "BGRA5551", "UV88",
	"UVWQ8888", "RGBA16161616F", "RGBA16161616", "UVLX8888", "R32F", "RGB323232F",
	"RGBA32323232F", "NV_DST16", "NV_DST24", "NV_INTZ", "NV_RAWZ", "ATI_DST16", "ATI_DST24",
	"NV_NULL", "ATI2N", "ATI1N",
}

// String returns the Source engine name of the format, e.g. "DXT1"
func (f Format) String() string {
	if int(f) < len(names) {
		return names[f]
	}
	if f == None {
		return "NONE"
	}
	return "Format(" + strconv.FormatUint(uint64(f), 10) + ")"
}
