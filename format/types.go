package format

// Colour: A pixel format that colour data is stored in
type Colour uint32

// RGBA (4bytes)
const RGBA8888 = Colour(0)

// ABGR (4bytes)
const ABGR8888 = Colour(1)

// RGB (3bytes)
const RGB888 = Colour(2)

// BGR (3bytes)
const BGR888 = Colour(3)

// RGB (2bytes)
const RGB565 = Colour(4)

// I8
const I8 = Colour(5)

// IA88
const IA88 = Colour(6)

// P8
const P8 = Colour(7)

// A8
const A8 = Colour(8)

// RGB (3bytes)
const RGB888_BLUESCREEN = Colour(9)

// RBGR (3bytes)
const BGR888_BLUESCREEN = Colour(10)

// ARGB (4bytes)
const ARGB8888 = Colour(11)

// BGRA (4bytes)
const BGRA8888 = Colour(12)

// Dxt1
const Dxt1 = Colour(13)

// Dxt3
const Dxt3 = Colour(14)

// Dxt5
const Dxt5 = Colour(15)

// BGRX - X may not be alpha (4bytes)
const BGRX8888 = Colour(16)

// BGR (2bytes)
const BGR565 = Colour(17)

// BGRX - X may not be alpha (2bytes)
const BGRX5551 = Colour(18)

// BGRA (2bytes)
const BGRA4444 = Colour(19)

// Dxt1 - with alpha
const Dxt1_OneBitAlpha = Colour(20)

// BGRA opacity either 0% or 100% (2bytes)
const BGRA5551 = Colour(21)

// UV (2bytes)
const UV88 = Colour(22)

// UV (4bytes)
const UVWQ8888 = Colour(23)

// RGBA (8bytes) as floats
const RGBA16161616F = Colour(24)

// RGBA (8bytes)
const RGBA16161616 = Colour(25)

// UVLX (8bytes)
const UVLX8888 = Colour(26)
