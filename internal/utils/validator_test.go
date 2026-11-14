package utils

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	testCases := []struct {
		email      string
		shouldPass bool
	}{
		{"test@example.com", true},
		{"invalid.email", false},
		{"test.name@example.com", true},
	}

	for _, tc := range testCases {
		result := ValidateEmail(tc.email)
		if result != tc.shouldPass {
			t.Errorf("For email %s, expected pass: %v, but got: %v", tc.email, tc.shouldPass, result)
		}
	}
}

func TestValidateURL(t *testing.T) {
	testCases := []struct {
		url        string
		shouldPass bool
	}{
		{"https://example.com", true},
		{"invalid.url", false},
		{"http://example.com/path?param=value", true},
		{"ftp://example.com", false},
	}

	for _, tc := range testCases {
		result := ValidateURL(tc.url)
		if result != tc.shouldPass {
			t.Errorf("For URL %s, expected pass: %v, but got: %v", tc.url, tc.shouldPass, result)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"blank becomes empty", "   ", ""},
		{"valid url is untouched", "https://example.com", "https://example.com"},
		{"valid url keeps path and query", "http://example.com/p?p=1", "http://example.com/p?p=1"},
		{"surrounding spaces are trimmed", "  https://example.com  ", "https://example.com"},
		// A bare host is what someone typing a homepage means.
		{"bare host gains a scheme", "example.com", "https://example.com"},
		{"bare host keeps its path", "www.example.com/path", "https://www.example.com/path"},
		{"bare host keeps its port", "example.com:8080", "https://example.com:8080"},
		{"bare host with unicode", "例子.中国", "https://例子.中国"},
		// Not recoverable, so the link is dropped instead of stored.
		{"other scheme is dropped", "ftp://example.com", ""},
		{"script url is dropped", "javascript:alert(1)", ""},
		{"relative path is dropped", "/relative/path", ""},
		{"protocol relative url is dropped", "//example.com", ""},
		{"free text is dropped", "not a url", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeURL(tc.in); got != tc.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestNormalizeURLRoundTrips pins the contract the callers rely on: whatever
// comes back is either empty or something ValidateURL accepts, so a stored link
// can always be sent back through the API.
func TestNormalizeURLRoundTrips(t *testing.T) {
	for _, in := range []string{
		"", "   ", "example.com", "https://example.com", "ftp://x.com",
		"javascript:alert(1)", "/rel", "//host", "b站", "a b", "https://", "http://",
	} {
		got := NormalizeURL(in)
		if got != "" && !ValidateURL(got) {
			t.Errorf("NormalizeURL(%q) = %q, which ValidateURL rejects", in, got)
		}
	}
}
