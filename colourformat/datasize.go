package colourformat


func GetImageSizeInBytes(index ColorFormat, width int, height int) int {
	switch index {
	case BGRX8888:
		return int(float32(width * height) * BytesPerPixel(index))
	case RGB888:
		return int(float32(width * height) * BytesPerPixel(index))
	case Dxt1:
		size := int(float32(width * height) * BytesPerPixel(index))
		if size < 8 {
			size = 8
		}
		return size
	case Dxt5:
		return int(float32(width * height) * BytesPerPixel(index))
	default:
		return 0
	}

	return 0
}

func BytesPerPixel(format ColorFormat) float32 {
	switch format {
	case BGRX8888:
		return 4
	case RGB888:
		return 3
	case Dxt1:
		return 0.5
	case Dxt5:
		return 1
	}

	return 0
}
