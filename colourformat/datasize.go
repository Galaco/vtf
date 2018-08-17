package colourformat


func GetLengthInBytesForFormat(index ColorFormat, width int, height int) int {
	switch index {
	case RGB888:
		return (width * height) * 3
	case Dxt1:
	case Dxt5:
		return (width * height * 3) / 2
	}

	return 0
}
