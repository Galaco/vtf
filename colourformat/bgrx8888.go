package colourformat

import (
	"errors"
)

func FromBGRX8888(width int, height int, buffer []byte) ([]byte, error) {
	ret := make([]byte, width * height * 3)

	if width * height * 3 > len(buffer) {
		return nil,errors.New("buffer is smaller than image dimensions")
	}

	// iterate pixel by pixel
	// ignores alpha, revered bgr to rgb
	for idx := 0; idx < len(ret); idx += 4 {
		ret[((idx / 4) * 3) + 0] = buffer[idx + 2]
		ret[((idx / 4) * 3) + 1] = buffer[idx + 1]
		ret[((idx / 4) * 3) + 2] = buffer[idx + 0]
	}

	return ret, nil
}