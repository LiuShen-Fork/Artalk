package utils

import (
	"net/url"
	"strings"

	"github.com/asaskevich/govalidator"
)

func ValidateEmail(email string) bool {
	return govalidator.IsEmail(email)
}

func ValidateURL(v string) bool {
	u, err := url.ParseRequestURI(v)
	if err != nil {
		return false
	}
	if len(u.Scheme) == 0 {
		return false
	}
	// Must https or http
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}
	return true
}

// NormalizeURL returns a link in the form that can be stored and later sent
// back through the API, or "" when it cannot be made valid.
//
// A bare host such as "example.com" gains an https:// scheme, because that is
// what someone typing a homepage means; everything else ValidateURL rejects is
// dropped rather than stored, which is what FindCreateUser already does. That
// keeps a value the update handlers would refuse out of the database, where it
// would come back with a 400 the next time the record is saved.
func NormalizeURL(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if ValidateURL(v) {
		return v
	}
	// Only a missing scheme is recoverable. Prepending to a value that already
	// carries one, or to a relative path, would invent a host.
	if !strings.Contains(v, "://") && !strings.HasPrefix(v, "/") {
		if candidate := "https://" + v; ValidateURL(candidate) {
			return candidate
		}
	}
	return ""
}
