package vtf

const (
	// FlagPointSampling
	FlagPointSampling = 0x0001
	// FlagTrilinearSampling
	FlagTrilinearSampling = 0x0002
	// FlagClampS
	FlagClampS = 0x0004
	// FlagClampT
	FlagClampT = 0x0008
	// FlagAnisotropicFiltering
	FlagAnisotropicFiltering = 0x0010
	// FlagHintDXT5
	FlagHintDXT5 = 0x0020
	// FlagPWLCorrected
	FlagPWLCorrected = 0x0040
	// FlagSRGB
	FlagSRGB = 0xFFFF // n/a
	// FlagNoCompress
	FlagNoCompress = 0x0040
	// FlagNormalMap
	FlagNormalMap = 0x0080
	// FlagNoMipmaps
	FlagNoMipmaps = 0x0100
	// FlagNoLevelOfDetail
	FlagNoLevelOfDetail = 0x0200
	// FlagNoMinimumMipmap
	FlagNoMinimumMipmap = 0x0400
	// FlagProcedural
	FlagProcedural = 0x0800
	// FlagOneBitAlpha
	FlagOneBitAlpha = 0x1000
	// FlagEightBitAlpha
	FlagEightBitAlpha = 0x2000
	// FlagEnvironmentMap
	FlagEnvironmentMap = 0x4000
	// FlagRenderTarget
	FlagRenderTarget = 0x8000
	// FlagDepthRenderTarget
	FlagDepthRenderTarget = 0x10000
	// FlagNoDebugOverride
	FlagNoDebugOverride = 0x20000
	// FlagSingleCopy
	FlagSingleCopy = 0x40000
	// FlagPreSRGB
	FlagPreSRGB = 0x80000
	// FlagOneOverMipmapLevelInAlpha
	FlagOneOverMipmapLevelInAlpha = 0x80000
	// FlagPreMultiplyColorByOneOverMipmapLevel
	FlagPreMultiplyColorByOneOverMipmapLevel = 0x100000
	// FlagNormalToDuDv
	FlagNormalToDuDv = 0x200000
	// FlagAlphaTestMipmapGeneration
	FlagAlphaTestMipmapGeneration = 0x400000
	// FlagNoDepthBuffer
	FlagNoDepthBuffer = 0x800000
	// FlagNiceFiltered
	FlagNiceFiltered = 0x1000000
	// FlagClampU
	FlagClampU = 0x2000000
	// FlagVertexTexture
	FlagVertexTexture = 0x4000000
	// FlagSSBump
	FlagSSBump = 0x8000000
	// FlagBorder
	FlagBorder = 0x20000000
)
