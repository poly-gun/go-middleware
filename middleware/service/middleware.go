package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "service-name"

// Options represents the configuration settings for the [Service] middleware component, including customizable service and header options.
type Options struct {
	// Name represents a string field in the [Options] struct. It is used to configure the service name in middleware configuration.
	Name string

	// Header represents an optional response-header to use to identify the handler's [Options.Name] key. Setting either the [Options.Header] or [Options.Name] to an empty string will prevent
	// the response from including the Header key-value. By default, the Header is set to "X-Service-Name". The associated Header's value can only be manually set via the
	// [Options.Name] value.
	Header string

	// Warnings specifies whether a warning log message should be logged in the [Service] middleware component's [Service.Handler] function. Defaults to true. Warnings are only emitted
	// if the [Options.Name] or [Options.Header] values contain an empty string, and therefore will skip updating any response header(s).
	Warnings bool
}

// Service represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Service struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Service] middleware's [Options] and returns the updated middleware instance.
func (s *Service) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if s.options == nil {
		s.options = &Options{
			Header:   "X-Service-Name",
			Name:     "",
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
func (s *Service) Handler(next http.Handler) http.Handler {
	s.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Update the request context with the applicable key-value pair(s).
		{
			ctx = context.WithValue(ctx, key, s.options.Name)
		}

		// Set the response headers according to the specification.
		{
			header := s.options.Header
			value := s.options.Name

			if header != "" && value != "" {
				w.Header().Set(http.CanonicalHeaderKey(header), value)
			} else if s.options.Warnings {
				slog.WarnContext(ctx, "Service-Name Middleware Configuration Contains Empty Value(s). Skipping Response Header(s)", slog.String("header", header), slog.String("value", value))
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Service] middleware, implementing [middleware.Configurable]. If [Service.Settings] isn't called,
// then the [Service.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Service)
}

// Value retrieves the service's name string from the provided context using a predefined key, or returns an empty string if the context is missing or invalid.
func Value(ctx context.Context) (service string) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(string); ok {
		service = v
	} else if test, valid := ctx.Value(t).(string); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		service = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))
	}

	return
}

// Runtime assurance that [Service] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Service)(nil)
