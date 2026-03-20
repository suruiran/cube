package cube

import "testing"

func TestWalkStream(t *testing.T) {
	for item, err := range Must(FsScanStream(t.Context(), "./", &WalkOptions{
		MatchPatterns: []string{"**/action/**/*.go"},
	})) {
		if err != nil {
			t.Log(err)
			continue
		}
		t.Log(item.String())
	}
}
