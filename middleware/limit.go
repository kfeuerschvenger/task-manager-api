package middleware

import "net/http"

// LimitBody caps every request body at maxBytes. Reads past the limit fail,
// so JSON decoding of an oversized body returns an error instead of buffering it.
func LimitBody(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}
