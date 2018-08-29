package colourformat


func GetImageSizeInBytes(index ColorFormat, width int, height int) int {
	size := int(float32(width * height) * BytesPerPixel(index))
	switch index {
	case Dxt1:
		if size < 8 {
			size = 8
		}
		return size
	}

	return size
}

func BytesPerPixel(format ColorFormat) float32 {
	switch format {
	case RGBA8888:
		return 4
	case ABGR8888:
		return 4
	case RGB888:
		return 3
	case BGR888:
		return 3
	case RGB565:
		return 2
	case I8:
		return 1
	case IA88:
		return 2
	case P8:
		return 1
	case A8:
		return 1
	case RGB888_BLUESCREEN:
		return 3
	case BGR888_BLUESCREEN:
		return 3
	case ARGB8888:
		return 4
	case BGRA8888:
		return 4
	case Dxt1:
		return 0.5
	case Dxt5:
		return 1
	case BGRX8888:
		return 4
	case BGR565:
		return 2
	case BGRX5551:
		return 2
	case BGRA4444:
		return 2
	case Dxt1_OneBitAlpha:
		return 0.5
	case BGRA5551:
		return 2
	case UV88:
		return 2
	case UVWQ8888:
		return 4
	case RGBA16161616F:
		return 8
	case RGBA16161616:
		return 8
	case UVLX8888:
		return 4
	}

	return 0
}
