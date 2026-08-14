package middleware

import (
	"fmt"
	"net/http"
	"regexp"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)
var roomNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)

func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 1-50 alphanumeric/underscore/dash characters")
	}
	return nil
}

func ValidateRoomName(name string) error {
	if !roomNameRegex.MatchString(name) {
		return fmt.Errorf("room name must be 1-50 alphanumeric/underscore/dash characters")
	}
	return nil
}

func MaxBytes(next http.HandlerFunc, maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next(w, r)
	}
}
