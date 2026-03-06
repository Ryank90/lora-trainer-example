package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func Auth(validKeys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(validKeys) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			key := strings.TrimPrefix(authHeader, "Bearer ")
			if key == authHeader {
				http.Error(w, `{"error":"invalid authorization format, expected Bearer token"}`, http.StatusUnauthorized)
				return
			}

			valid := false
			for _, k := range validKeys {
				if subtle.ConstantTimeCompare([]byte(key), []byte(k)) == 1 {
					valid = true
					break
				}
			}

			if !valid {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
