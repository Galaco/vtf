package vtf

type header struct {
	headerCommon
	header72
	header73
}

type headerCommon struct {
	Signature [4]byte						//File signature char
	Version [2]uint32						//Version[0].version[1] e.g. 7.2 uint
	HeaderSize uint32						//Size of header (16 byte aligned, currently 80bytes) uint
	Width uint16							//Width of largest mipmap (^2) ushort
	Height uint16							//Height of largest mipmap (^2) ushort
	Flags uint32							//VTF Flags uint
	Frames uint16							//Number of frames (if animated) default: 1 ushort
	FirstFrame uint16						//First frame in animation (0 based) ushort
	_ [4]byte
	Reflectivity [3]float32					//reflectivity vector float
	_ [4]byte
	BumpmapScale float32					//Bumpmap scale float
	HighResImageFormat uint32				//High resolution image format uint (probably 4?)
	MipmapCount uint8						//Number of mipmaps uchar
	LowResImageFormat uint32				//Low resolution image format (always DXT1 [=14]) uint
	LowResImageWidth uint8					//Low resolution image width uchar
	LowResImageHeight uint8					//Low resolution image height uchar
}

type header72 struct {
	Depth uint16							//Depth of the largest mipmap in pixels (^2) ushort
}

type header73 struct {
	_ [3]byte
	NumResource uint32						// Number of resources this vtf has
}