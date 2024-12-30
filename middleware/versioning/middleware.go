package versioning

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "versioning"

// Options represents the configuration settings for the [Versioning] middleware component, including customizable server and header options.
type Options struct {
	// API specifies the API version or identifier used by the [Versioning] middleware component.
	API string

	// Service represents the version of the service associated with the [Versioning] middleware component.
	Service string

	// Warnings specifies whether a warning log message should be logged in the [Versioning] middleware component's [Versioning.Handler] function. Defaults to false.
	Warnings bool
}

type Versions struct {
	API     string `json:"api"`
	Service string `json:"service"`
}

// Versioning represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Versioning struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Versioning] middleware's [Options] and returns the updated middleware instance.
func (v *Versioning) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if v.options == nil {
		v.options = &Options{
			API:      "",
			Service:  "",
			Warnings: false,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(v.options)
		}
	}

	return v
}

// Handler applies middleware settings to modify the request context and set response headers. It forwards the request to the next handler in the chain.
func (v *Versioning) Handler(next http.Handler) http.Handler {
	v.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if value := r.Header.Get(http.CanonicalHeaderKey("X-API-Version")); value != "" {
			v.options.API = value
		}

		// Update the request context with the applicable key-value pair(s).
		{
			ctx = context.WithValue(ctx, key, &Versions{
				API:     v.options.API,
				Service: v.options.Service,
			})
		}

		// Evaluate the API version.
		if value := v.options.API; value != "" {
			w.Header().Set("X-API-Version", value)
		} else if v.options.Warnings {
			slog.WarnContext(ctx, "Versioning Middleware Configuration Contains Empty Value(s). Skipping Response Header(s)", slog.String("header", "X-API-Version"), slog.String("value", value))
		}

		// Evaluate the Service version.
		if value := v.options.Service; value != "" {
			w.Header().Set("X-Service-Version", value)
		} else if v.options.Warnings {
			slog.WarnContext(ctx, "Versioning Middleware Configuration Contains Empty Value(s). Skipping Response Header(s)", slog.String("header", "X-Service-Version"), slog.String("value", value))
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Versioning] middleware, implementing [middleware.Configurable]. If [Versions.Settings] isn't called,
// then the [Versions.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Versioning)
}

// Value retrieves the [Versions] from the provided context using a predefined key, or returns a nil value if the middleware isn't enabled.
func Value(ctx context.Context) (versions *Versions) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(*Versions); ok {
		versions = v
	} else if test, valid := ctx.Value(t).(*Versions); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		versions = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))
	}

	return
}

// Runtime assurance that [Versioning] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Versioning)(nil)
