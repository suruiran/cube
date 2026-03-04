package cube

import "testing"

func TestRandAscii(t *testing.T) {
	ascii := RandAsciiBytes(6)
	t.Logf("ascii: %s", ascii)
}
