package format

import "testing"

func TestFormat_String(t *testing.T) {
	tests := map[Format]string{
		RGBA8888:   "RGBA8888",
		Dxt1:       "DXT1",
		Ati2n:      "ATI2N",
		Ati1n:      "ATI1N",
		None:       "NONE",
		Format(99): "Format(99)",
	}
	for f, want := range tests {
		if got := f.String(); got != want {
			t.Errorf("Format(%d).String(): got %q, want %q", uint32(f), got, want)
		}
	}
}
