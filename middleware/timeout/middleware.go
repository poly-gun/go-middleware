package timeout

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "timeout"

const defaultTimeoutDuration = time.Second * 30

// Options defines configurable settings for timeout behaviors, including response header customization and operation timeout durations.
type Options struct {
	// Timeout represents the duration to wait before considering an operation as timed out. If unspecified, or a negative value,
	// a default of 30 seconds is overwritten.
	Timeout time.Duration

	// Header represents an optional response-header key. Setting the [Options.Header] to an empty string will prevent
	// the response from including the Header key-value. By default, the Header is set to "X-Timeout".
	Header string
}

// Timeout represents a middleware component that applies configurable timeout settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Timeout struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Timeout] middleware's [Options] and returns the updated middleware instance.
func (t *Timeout) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if t.options == nil {
		t.options = &Options{
			Header:  "X-Timeout",
			Timeout: defaultTimeoutDuration,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(t.options)
		}
	}

	// Ensure user-provided configuration is compliant with the middleware's expectations.
	if t.options.Timeout <= 0 {
		slog.Warn("Invalid Timeout Value Specified - Using Default Duration")

		t.options.Timeout = defaultTimeoutDuration
	}

	return t
}

// Handler applies timeout middleware to the provided HTTP handler, enforcing a request timeout and adding optional timeout metadata to the response.
func (t *Timeout) Handler(next http.Handler) http.Handler {
	t.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Update the request context with the applicable key-value pair(s).
		{
			ctx = context.WithValue(ctx, key, t.options.Timeout)
		}

		// Set the response headers according to the specification.
		{
			if t.options.Header != "" {
				value := t.options.Timeout.String()

				w.Header().Set(http.CanonicalHeaderKey(t.options.Header), value)
			}
		}

		ctx, cancel := context.WithTimeout(ctx, t.options.Timeout)
		defer func() {
			cancel()
			e := ctx.Err()
			if errors.Is(e, context.DeadlineExceeded) {
				http.Error(w, "gateway-timeout", http.StatusGatewayTimeout)
				return
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Timeout] middleware, implementing [middleware.Configurable]. If [Timeout.Settings] isn't called,
// then the [Timeout.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Timeout)
}

// Value retrieves a [time.Duration] from the provided context using a predefined key or returns a default timeout if the key's value is missing or invalid.
func Value(ctx context.Context) (duration time.Duration) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(time.Duration); ok {
		duration = v
	} else if test, valid := ctx.Value(t).(time.Duration); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		duration = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))

		return defaultTimeoutDuration
	}

	if duration <= 0 {
		slog.WarnContext(ctx, "Invalid Duration Value Specified - Using Default Duration", slog.String("error", "Invalid-Duration-Value"))

		return defaultTimeoutDuration
	}

	return
}

// Runtime assurance that [Timeout] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Timeout)(nil)
