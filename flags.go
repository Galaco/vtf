package vtf

// Flags is a set of texture flags
type Flags uint32

// Has reports whether every flag in f is set
func (flags Flags) Has(f Flags) bool {
	return flags&f == f
}

// Texture flags, as stored in Header.Flags.
// Some bits have been reused across engine branches, so several names share a value.
const (
	// FlagPointSampling disables filtering
	FlagPointSampling Flags = 0x0001
	// FlagTrilinearSampling enables trilinear filtering
	FlagTrilinearSampling Flags = 0x0002
	// FlagClampS clamps S (U) coordinates
	FlagClampS Flags = 0x0004
	// FlagClampT clamps T (V) coordinates
	FlagClampT Flags = 0x0008
	// FlagAnisotropicFiltering enables anisotropic filtering
	FlagAnisotropicFiltering Flags = 0x0010
	// FlagHintDXT5 hints that the texture should be compressed as DXT5
	FlagHintDXT5 Flags = 0x0020
	// FlagSRGB marks the texture as sRGB (Source 2013). Shares a value with FlagPWLCorrected and FlagNoCompress
	FlagSRGB Flags = 0x0040
	// FlagPWLCorrected marks the texture as PWL corrected (later engine branches). Shares a value with FlagSRGB
	FlagPWLCorrected Flags = 0x0040
	// FlagNoCompress is deprecated. Shares a value with FlagSRGB
	FlagNoCompress Flags = 0x0040
	// FlagNormalMap marks the texture as a normal map
	FlagNormalMap Flags = 0x0080
	// FlagNoMipmaps disables mipmapping
	FlagNoMipmaps Flags = 0x0100
	// FlagNoLevelOfDetail excludes the texture from texture level of detail settings
	FlagNoLevelOfDetail Flags = 0x0200
	// FlagNoMinimumMipmap loads every mipmap, ignoring the minimum mipmap setting
	FlagNoMinimumMipmap Flags = 0x0400
	// FlagProcedural marks the texture as procedurally generated
	FlagProcedural Flags = 0x0800
	// FlagOneBitAlpha is set when alpha is either fully opaque or transparent
	FlagOneBitAlpha Flags = 0x1000
	// FlagEightBitAlpha is set when the texture has an 8 bit alpha channel
	FlagEightBitAlpha Flags = 0x2000
	// FlagEnvironmentMap marks the texture as a cubemap
	FlagEnvironmentMap Flags = 0x4000
	// FlagRenderTarget marks the texture as a render target
	FlagRenderTarget Flags = 0x8000
	// FlagDepthRenderTarget marks the texture as a depth render target
	FlagDepthRenderTarget Flags = 0x10000
	// FlagNoDebugOverride prevents debug overrides of the texture
	FlagNoDebugOverride Flags = 0x20000
	// FlagSingleCopy loads the texture once, shared between materials
	FlagSingleCopy Flags = 0x40000
	// FlagPreSRGB marks the texture as pre-sRGB (later engine branches). Shares a value with FlagOneOverMipmapLevelInAlpha
	FlagPreSRGB Flags = 0x80000
	// FlagOneOverMipmapLevelInAlpha is deprecated. Shares a value with FlagPreSRGB
	FlagOneOverMipmapLevelInAlpha Flags = 0x80000
	// FlagPreMultiplyColorByOneOverMipmapLevel is deprecated
	FlagPreMultiplyColorByOneOverMipmapLevel Flags = 0x100000
	// FlagNormalToDuDv is deprecated
	FlagNormalToDuDv Flags = 0x200000
	// FlagAlphaTestMipmapGeneration is deprecated
	FlagAlphaTestMipmapGeneration Flags = 0x400000
	// FlagNoDepthBuffer disables the depth buffer of a render target
	FlagNoDepthBuffer Flags = 0x800000
	// FlagNiceFiltered is deprecated
	FlagNiceFiltered Flags = 0x1000000
	// FlagClampU clamps U (R) coordinates of volume textures
	FlagClampU Flags = 0x2000000
	// FlagVertexTexture marks the texture as usable from vertex shaders
	FlagVertexTexture Flags = 0x4000000
	// FlagSSBump marks the texture as a self shadowing bump map
	FlagSSBump Flags = 0x8000000
	// FlagUnfilterableOK is deprecated
	FlagUnfilterableOK Flags = 0x10000000
	// FlagBorder clamps to the border colour
	FlagBorder Flags = 0x20000000
	// FlagSpecVarRed is deprecated
	FlagSpecVarRed Flags = 0x40000000
	// FlagSpecVarAlpha is deprecated
	FlagSpecVarAlpha Flags = 0x80000000
)
