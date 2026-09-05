package pokedex

import (
	"net/http/httptest"
	"testing"
)

func TestRequireName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantName string
		wantOK   bool
	}{
		{name: "present", url: "/species?name=pikachu", wantName: "pikachu", wantOK: true},
		{name: "trims whitespace", url: "/species?name=%20pikachu%20", wantName: "pikachu", wantOK: true},
		{name: "missing", url: "/species", wantName: "", wantOK: false},
		{name: "empty", url: "/species?name=", wantName: "", wantOK: false},
		{name: "whitespace only", url: "/species?name=%20%20", wantName: "", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tc.url, nil)
			name, ok := requireName(r)
			if name != tc.wantName || ok != tc.wantOK {
				t.Errorf("requireName() = (%q, %v), want (%q, %v)", name, ok, tc.wantName, tc.wantOK)
			}
		})
	}
}

func TestLikePattern(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "meowth", want: "%meowth%"},
		{name: "escapes percent", in: "50%", want: `%50\%%`},
		{name: "escapes underscore", in: "poke_ball", want: `%poke\_ball%`},
		{name: "escapes backslash", in: `a\b`, want: `%a\\b%`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := likePattern(tc.in); got != tc.want {
				t.Errorf("likePattern(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
