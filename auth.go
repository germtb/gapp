package gapp

import (
	"context"
	"net/http"
)

type authTokenKeyType struct{}

var authTokenKey = authTokenKeyType{}

// SetAuthToken returns a new request with the given token stored in its context.
func SetAuthToken(r *http.Request, token any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authTokenKey, token))
}

// GetAuthToken retrieves the auth token from the request context.
// Returns nil if no token has been set.
func GetAuthToken(r *http.Request) any {
	return r.Context().Value(authTokenKey)
}

// AuthMiddleware creates a middleware that validates requests using the provided function.
// If validate returns a non-nil token, it's stored in the request context via SetAuthToken.
// If validate returns nil, the request proceeds without a token (unauthenticated).
// This allows handlers to decide individually whether auth is required.
func AuthMiddleware(validate func(r *http.Request) any) Middleware {
	return func(next RpcHandler) RpcHandler {
		return func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error) {
			if token := validate(r); token != nil {
				r = SetAuthToken(r, token)
			}
			return next(w, r, method, body)
		}
	}
}

// RequireAuth wraps a UnaryHandler to reject unauthenticated requests with 401.
// Use this on individual handlers that require authentication.
func RequireAuth(handler UnaryHandler) UnaryHandler {
	return func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error) {
		if GetAuthToken(r) == nil {
			return nil, ErrUnauthenticated("authentication required")
		}
		return handler(w, r, method, body)
	}
}
