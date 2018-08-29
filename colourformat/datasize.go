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
	case BGRX8888:
		return 4
	case BGR888:
		return 3
	case RGB888:
		return 3
	case Dxt1:
		return 0.5
	case Dxt5:
		return 1
	}

	return 0
}
