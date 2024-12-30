package middleware

import (
	"net/http"
)

// Configurable defines an interface for applying configurable behaviors to HTTP handlers using generic Options settings.
type Configurable[Options interface{}] interface {
	// Handler wraps the provided [http.Handler] with middleware functionality and returns a new [http.Handler].
	Handler(next http.Handler) http.Handler

	// Settings applies configuration functions to the middleware's options and returns the updated middleware.
	Settings(...func(o *Options)) Configurable[Options]
}

// Middleware represents a structure to manage a chain of HTTP middleware functions.
// It wraps and applies middleware to an [http.Handler] in order of addition.
type Middleware struct {
	middleware []func(http.Handler) http.Handler
}

// Add appends one or more middleware functions to the middleware chain in the order they are provided.
func (m *Middleware) Add(middleware ...func(http.Handler) http.Handler) {
	if length := len(middleware); length == 0 {
		return
	}

	m.middleware = append(m.middleware, middleware...)
}

// Handler applies the middleware chain to the provided parent [http.Handler] and returns the final wrapped handler.
// If no middleware is present, the parent handler is returned as is.
func (m *Middleware) Handler(parent http.Handler) (handler http.Handler) {
	if length := len(m.middleware); length == 0 {
		return parent
	}

	// Wrap the final handler with the middleware chain.
	handler = m.middleware[len(m.middleware)-1](parent)
	for i := len(m.middleware) - 2; i >= 0; i-- {
		handler = m.middleware[i](handler)
	}

	return
}

// New initializes and returns a pointer to a new [Middleware] instance.
func New() *Middleware {
	return new(Middleware)
}
