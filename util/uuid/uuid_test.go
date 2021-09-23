package uuid

import (
	"testing"
)

func TestParseDB(t *testing.T) {
	testCases := []struct {
		desc string
		in   string
		out  UUID
	}{
		{
			desc: "zero",
			in:   "00000000-0000-0000-0000-000000000000",
		},
		{
			desc: "random",
			in:   "6fa459ea-ee8a-3ca4-894e-db77e160355e",
			out:  UUID{0x6f, 0xa4, 0x59, 0xea, 0xee, 0x8a, 0x3c, 0xa4, 0x89, 0x4e, 0xdb, 0x77, 0xe1, 0x60, 0x35, 0x5e},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if MustParseWithHyphens(tC.in) != tC.out {
				t.Errorf("expected %s, got %s", tC.out, MustParseWithHyphens(tC.in))
			}
		})
	}
}

func BenchmarkNew(b *testing.B) {
	var f UUID
	for i := 0; i < b.N; i++ {
		f = New()
	}
	_ = f
}
