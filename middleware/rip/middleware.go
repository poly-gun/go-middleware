package rip

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "real-ip"

const (
	trueClientIP  = "True-Client-IP"
	xForwardedFor = "X-Forwarded-For"
	xRealIP       = "X-Real-IP"
)

// Options represents the configuration settings for the [Server] middleware component.
type Options struct {
	// Level specifies whether a log message should be logged in the [Server] middleware component's [Server.Handler] function. Default is nil. A value of nil
	// causes the [Server.Handler] to skip logging of the ip-related header(s), entirely. See the [slog.Leveler] interface for additional information.
	Level slog.Leveler
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
			Level: nil,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(s.options)
		}
	}

	return s
}

// Handler applies middleware settings to modify the request context. It forwards the request to the next handler in the chain.
func (s *Server) Handler(next http.Handler) http.Handler {
	s.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var value string

		switch {
		case r.Header.Get(trueClientIP) != "":
			value = r.Header.Get(trueClientIP)
		case r.Header.Get(xForwardedFor) != "":
			value = r.Header.Get(xForwardedFor)
		case r.Header.Get(xRealIP) != "":
			value = r.Header.Get(xRealIP)
		}

		if strings.Contains(value, ",") {
			values := strings.Split(value, ",")

			value = values[0]
		}

		if v := s.options.Level; v != nil && value != "" {
			slog.Log(ctx, v.Level(), "X-Real-IP Middleware", slog.String("value", value))
		}

		// Store user agent in the context.
		ctx = context.WithValue(ctx, key, value)

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
