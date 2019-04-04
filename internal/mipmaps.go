package internal

import (
	"github.com/galaco/vtf/format"
	"math"
)

// ComputeMipmapSizes computes all mipmap sizes
func ComputeMipmapSizes(num int, width int, height int) [][2]int {
	mipmaps := make([][2]int, num)

	for i := num - 1; i >= 0; i-- {
		mipmaps[i] = [2]int{width, height}

		width = int(math.Ceil(float64(width) / 2))
		height = int(math.Ceil(float64(height) / 2))
	}

	return mipmaps
}

// ComputeSizeOfMipmapData returns the size in bytes
func ComputeSizeOfMipmapData(width int, height int, storedFormat format.Format) int {
	// Supported compressed formats must be at least 4x4.
	// The format pads smaller sizes out to 4
	if isCompressedFormat(storedFormat) {
		if width < 4 {
			width = 4
		}
		if height < 4 {
			height = 4
		}
	}

	return int(bytesPerPixel(storedFormat) * float32(width) * float32(height))
}

// isCompressedFormat determines whether the provided format
// is a compressed format. Supported compressed formats are Dxt* only
func isCompressedFormat(storedFormat format.Format) bool {
	if storedFormat == format.Dxt1 ||
		storedFormat == format.Dxt3 ||
		storedFormat == format.Dxt5 {
		return true
	}

	return false
}

// bytesPerPixel gets number of bytes required to represent a single pixel
// May be non-integer for compressed formats
func bytesPerPixel(storedFormat format.Format) float32 {
	switch storedFormat {
	case format.RGBA8888:
		return 4
	case format.ABGR8888:
		return 4
	case format.RGB888:
		return 3
	case format.BGR888:
		return 3
	case format.RGB565:
		return 2
	case format.I8:
		return 1
	case format.IA88:
		return 2
	case format.P8:
		return 1
	case format.A8:
		return 1
	case format.RGB888BLUESCREEN:
		return 3
	case format.BGR888BLUESCREEN:
		return 3
	case format.ARGB8888:
		return 4
	case format.BGRA8888:
		return 4
	case format.Dxt1:
		return 0.5
	case format.Dxt5:
		return 1
	case format.BGRX8888:
		return 4
	case format.BGR565:
		return 2
	case format.BGRX5551:
		return 2
	case format.BGRA4444:
		return 2
	case format.Dxt1OneBitAlpha:
		return 0.5
	case format.BGRA5551:
		return 2
	case format.UV88:
		return 2
	case format.UVWQ8888:
		return 4
	case format.RGBA16161616F:
		return 8
	case format.RGBA16161616:
		return 8
	case format.UVLX8888:
		return 4
	}

	return 0
}
