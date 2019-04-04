package format

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
	// I8
	I8 = Format(5)
	// IA88
	IA88 = Format(6)
	// P8
	P8 = Format(7)
	// A8
	A8 = Format(8)
	// RGB888BLUESCREEN RGB (3bytes)
	RGB888BLUESCREEN = Format(9)
	// BGR888BLUESCREEN RBGR (3bytes)
	BGR888BLUESCREEN = Format(10)
	// ARGB8888 ARGB (4bytes)
	ARGB8888 = Format(11)
	// BGRA8888 BGRA (4bytes)
	BGRA8888 = Format(12)
	// Dxt1
	Dxt1 = Format(13)
	// Dxt3
	Dxt3 = Format(14)
	// Dxt5
	Dxt5 = Format(15)
	// BGRX8888 BGRX - X may not be alpha (4bytes)
	BGRX8888 = Format(16)
	// BGR565 BGR (2bytes)
	BGR565 = Format(17)
	// BGRX5551 BGRX - X may not be alpha (2bytes)
	BGRX5551 = Format(18)
	// BGRA4444 BGRA (2bytes)
	BGRA4444 = Format(19)
	// Dxt1OneBitAlpha Dxt1 - with alpha
	Dxt1OneBitAlpha = Format(20)
	// BGRA5551 BGRA opacity either 0% or 100% (2bytes)
	BGRA5551 = Format(21)
	// UV88 UV (2bytes)
	UV88 = Format(22)
	// UVWQ8888 UV (4bytes)
	UVWQ8888 = Format(23)
	// RGBA16161616F RGBA (8bytes) as floats
	RGBA16161616F = Format(24)
	// RGBA16161616 RGBA (8bytes)
	RGBA16161616 = Format(25)
	// UVLX8888 UVLX (8bytes)
	UVLX8888 = Format(26)
)
