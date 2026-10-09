package internal

import (
	"errors"

	"github.com/galaco/vtf/format"
)

// ErrUnsupportedFormat is returned when the size of a format's data cannot be computed
var ErrUnsupportedFormat = errors.New("unsupported image format")

// MipmapDimensions returns the dimensions of a mipmap level, where level 0 is the
// full size image. Each axis halves per level, to a minimum of 1.
func MipmapDimensions(width, height, depth, level int) (int, int, int) {
	return shrink(width, level), shrink(height, level), shrink(depth, level)
}

func shrink(size, level int) int {
	size >>= level
	if size < 1 {
		return 1
	}
	return size
}

// ComputeSizeOfMipmapData returns the size in bytes of a single width x height image
// stored in storedFormat.
func ComputeSizeOfMipmapData(width int, height int, storedFormat format.Format) (int, error) {
	if blockSize := blockSize(storedFormat); blockSize > 0 {
		// Block compressed formats store 4x4 pixel blocks, so dimensions round up to 4
		return ((width + 3) / 4) * ((height + 3) / 4) * blockSize, nil
	}

	bpp := bytesPerPixel(storedFormat)
	if bpp == 0 {
		return 0, ErrUnsupportedFormat
	}

	return width * height * bpp, nil
}

// blockSize returns the bytes per 4x4 block of a block compressed format,
// or 0 if the format is not block compressed.
func blockSize(storedFormat format.Format) int {
	switch storedFormat {
	case format.Dxt1, format.Dxt1OneBitAlpha, format.Ati1n:
		return 8
	case format.Dxt3, format.Dxt5, format.Ati2n:
		return 16
	}

	return 0
}

// bytesPerPixel gets number of bytes required to represent a single pixel
// of an uncompressed format, or 0 if the format is not supported.
func bytesPerPixel(storedFormat format.Format) int {
	switch storedFormat {
	case format.I8, format.P8, format.A8:
		return 1
	case format.RGB565, format.BGR565, format.BGRX5551, format.BGRA4444, format.BGRA5551,
		format.IA88, format.UV88:
		return 2
	case format.RGB888, format.BGR888, format.RGB888BLUESCREEN, format.BGR888BLUESCREEN:
		return 3
	case format.RGBA8888, format.ABGR8888, format.ARGB8888, format.BGRA8888, format.BGRX8888,
		format.UVWQ8888, format.UVLX8888, format.R32F:
		return 4
	case format.RGBA16161616F, format.RGBA16161616:
		return 8
	case format.RGB323232F:
		return 12
	case format.RGBA32323232F:
		return 16
	}

	// Depth buffer formats are render target only and never stored on disk
	return 0
}
