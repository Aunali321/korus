package korus

import "testing"

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"korus.example.com", "https://korus.example.com"},
		{" https://korus.example.com/ ", "https://korus.example.com"},
		{"https://korus.example.com/api", "https://korus.example.com"},
		{"https://korus.example.com/api/", "https://korus.example.com"},
		{"http://192.168.1.5:8080", "http://192.168.1.5:8080"},
		{"https://host/korus/api?x=1#y", "https://host/korus"},
	}
	for _, tc := range cases {
		got, err := NormalizeURL(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}

	for _, in := range []string{"", "   ", "ftp://host", "https://"} {
		if got, err := NormalizeURL(in); err == nil {
			t.Errorf("NormalizeURL(%q) = %q; want an error", in, got)
		}
	}
}
