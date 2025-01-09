package useragent

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "user-agent"

// Options represents the configuration settings for the [Server] middleware component.
type Options struct {
	// Warnings specifies whether a warning log message should be logged in the [Server] middleware component's [Server.Handler] function. Defaults to true. Warnings are only emitted
	// if the user-agent cannot be found in the request headers.
	Warnings bool
}

// Server represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Server struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Server] middleware's [Options] and returns the updated middleware instance.
func (s *Server) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if s.options == nil {
		s.options = &Options{
			Warnings: true,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(s.options)
		}
	}

	return s
}

// Handler applies middleware settings to modify the request context and set response headers. It forwards the request to the next handler in the chain.
func (s *Server) Handler(next http.Handler) http.Handler {
	s.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract user agent from the request header.
		ua := r.Header.Get("User-Agent")

		if ua == "" && s.options.Warnings {
			slog.WarnContext(ctx, "User-Agent Header Not Found", slog.String("value", ua))
		}

		// Store user agent in the context.
		ctx = context.WithValue(ctx, key, ua)

		// Pass the request along with the new context.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Server] middleware, implementing [middleware.Configurable]. If [Server.Settings] isn't called,
// then the [Server.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Server)
}

// Value retrieves context value for the following package's middleware.
func Value(ctx context.Context) (agent string) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(string); ok {
		agent = v
	} else if test, valid := ctx.Value(t).(string); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		agent = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))
	}

	return
}

// Runtime assurance that [Server] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Server)(nil)
