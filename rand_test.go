package cube

import "testing"

func TestRandAscii(t *testing.T) {
	code := RandChoices(AsciiChars, 6)
	t.Logf("ascii: %s", code)

	code, err := RandBytesCrypto(AsciiChars, 6)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ascii: %s", code)
}

func BenchmarkRandAscii(b *testing.B) {
	for b.Loop() {
		RandChoices(AsciiChars, 6)
	}
}

func BenchmarkRandBytesCrypto(b *testing.B) {
	for b.Loop() {
		_, _ = RandBytesCrypto(AsciiChars, 6)
	}
}
