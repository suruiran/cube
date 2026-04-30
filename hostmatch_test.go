package cube

import (
	"testing"

	"github.com/suruiran/cube/seqs"
)

func TestIsHostMatch(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		whitehosts []string
		want       bool
	}{
		{
			name:       "empty",
			host:       "",
			whitehosts: []string{},
			want:       false,
		},
		{
			name:       "no_match",
			host:       "google.com",
			whitehosts: []string{"*.example.com", "api.github.com"},
			want:       false,
		},
		{
			name:       "localhost",
			host:       "localhost",
			whitehosts: []string{"localhost"},
			want:       true,
		},
		{
			name:       "uppercase_host",
			host:       "API.GITHUB.COM",
			whitehosts: []string{"api.github.com"},
			want:       true,
		},
		{
			name:       "dirty_config_uppercase",
			host:       "api.github.com",
			whitehosts: []string{"API.GITHUB.COM"},
			want:       true,
		},

		{
			name:       "wildcard_match",
			host:       "sub.example.com",
			whitehosts: []string{"*.example.com"},
			want:       true,
		},
		{
			name:       "multi_level_subdomain_match",
			host:       "a.b.example.com",
			whitehosts: []string{"*.example.com"},
			want:       true,
		},
		{
			name:       "dirty_config_multi_wildcard_strip",
			host:       "sub.example.com",
			whitehosts: []string{"*.*.example.com"},
			want:       true,
		},

		{
			name:       "suffix_attack",
			host:       "fakeexample.com",
			whitehosts: []string{"*.example.com"},
			want:       false,
		},
		{
			name:       "prefix_attack",
			host:       "example.com.evil.com",
			whitehosts: []string{"*.example.com"},
			want:       false,
		},
		{
			name:       "exact_match",
			host:       "sub.example.com",
			whitehosts: []string{"example.com"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHostMatch(tt.host, seqs.FromSlice(tt.whitehosts)); got != tt.want {
				t.Errorf("\nName: %s\nInput Host: %q\nWhitehosts: %v\nWant (want): %v\nGot (got): %v",
					tt.name, tt.host, tt.whitehosts, tt.want, got)
			}
		})
	}
}
