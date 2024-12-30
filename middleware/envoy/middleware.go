package envoy

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
const key keyer = "envoy"

// Options represents the configuration settings for the [Envoy] middleware component.
type Options struct {
	// Debug specifies whether a request containing envoy-related proxy headers will include log message(s). Defaults to true.
	Debug bool
}

// Envoy represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Envoy struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Envoy] middleware's [Options] and returns the updated middleware instance.
func (e *Envoy) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if e.options == nil {
		e.options = &Options{
			Debug: false,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(e.options)
		}
	}

	return e
}

// Handler applies middleware settings to modify the request context and set response headers. It forwards the request to the next handler in the chain.
func (e *Envoy) Handler(next http.Handler) http.Handler {
	e.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		headers := http.Header{}
		for k, v := range r.Header {
			if strings.HasPrefix(strings.ToLower(k), "x-envoy-") {
				for index := range v {
					headers.Add(k, v[index])
				}
			}
		}

		if e.options.Debug { // For unit-testing purposes, it's important that only one log message is reported by slog.
			if headers != nil && len(headers) > 0 {
				slog.DebugContext(ctx, "Envoy Proxy Request Header(s)", slog.Any("headers", headers))
			} else {
				slog.DebugContext(ctx, "No Envoy Proxy Request Header(s)", slog.Any("headers", headers))
			}
		}

		// Update the request context with the applicable key-value pair(s).
		{
			ctx = context.WithValue(ctx, key, &headers)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Envoy] middleware, implementing [middleware.Configurable]. If [Envoy.Settings] isn't called,
// then the [Envoy.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Envoy)
}

// Value retrieves a [http.Header] pointer representing the envoy proxy's related headers. If a nil value is returned, it can be
// assumed that the [Envoy] middleware isn't enabled for the particular caller's chain. If the value is an empty map,
// it's to be assumed the [Envoy] middleware is enabled, however, no envoy-related proxy headers were found.
func Value(ctx context.Context) (headers *http.Header) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(*http.Header); ok {
		headers = v
	} else if test, valid := ctx.Value(t).(*http.Header); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		headers = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))
	}

	return
}

// Runtime assurance that [Envoy] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Envoy)(nil)
