package cors

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	external "github.com/rs/cors"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "cors"

// Options represents the configuration settings for the [CORS] middleware component.
type Options struct {
	// Debug represents a boolean flag to enable debug-related logging. Defaults to false.
	Debug bool
}

// CORS represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type CORS struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Service] middleware's [Options] and returns the updated middleware instance.
func (c *CORS) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if c.options == nil {
		c.options = &Options{
			Debug: false,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(c.options)
		}
	}

	return c
}

// Handler is a middleware method that wraps the provided [http.Handler], applying [CORS] settings and injecting context with predefined values.
func (c *CORS) Handler(next http.Handler) http.Handler {
	c.Settings() // Ensure the options field isn't nil.

	internals := external.Options{
		AllowedOrigins:             nil,
		AllowOriginFunc:            func(origin string) bool { return true },
		AllowOriginVaryRequestFunc: nil,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:       []string{"*"},
		ExposedHeaders:       []string{"*"},
		MaxAge:               300, // Maximum value not ignored by any of major browsers
		AllowCredentials:     true,
		AllowPrivateNetwork:  true,
		OptionsPassthrough:   false,
		OptionsSuccessStatus: http.StatusNoContent,
		Debug:                c.options.Debug,
		Logger:               nil,
	}

	wrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		{
			value := true

			ctx = context.WithValue(ctx, key, value)
		}

		{
			switch {
			case w.Header().Get("Access-Control-Allow-Headers") == "":
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(internals.AllowedHeaders, ", "))
				fallthrough
			case w.Header().Get("Access-Control-Allow-Methods") == "":
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(internals.AllowedMethods, ", "))
				fallthrough
			case w.Header().Get("Access-Control-Expose-Headers") == "":
				w.Header().Set("Access-Control-Expose-Headers", "*")
				fallthrough
			default:
				// ...
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})

	if c.options.Debug {
		slog.Debug("Instantiating CORS Handler")
	}

	handle := external.New(internals)

	return handle.Handler(wrapper)
}

// New creates a new instance of the [CORS] middleware, implementing [middleware.Configurable]. If [CORS.Settings] isn't called,
// then the [CORS.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(CORS)
}

// Value retrieves a boolean value from the provided context, indicating if the [CORS] middleware is enabled, based on predefined context keys, and logs warnings for invalid or missing key evaluation.
func Value(ctx context.Context) (enabled bool) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(bool); ok {
		enabled = v
	} else if test, valid := ctx.Value(t).(bool); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		enabled = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))
	}

	return
}

// Runtime assurance that [CORS] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*CORS)(nil)
