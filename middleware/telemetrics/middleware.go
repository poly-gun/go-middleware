package telemetrics

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "telemetrics"

// merge accepts any number of []string arguments and returns a slice of unique strings.
func merge(slices ...[]string) []string {
	unique := make(map[string]bool)
	for _, slice := range slices {
		for _, str := range slice {
			unique[str] = true
		}
	}

	// Convert map keys back to a slice
	result := make([]string, 0, len(unique))
	for str := range unique {
		result = append(result, str)
	}

	return result
}

// remove removes any string from "source" that exists in "removals".
func remove(source []string, removals []string) []string {
	// First, convert "removals" into a set (map with empty struct values).
	negations := make(map[string]struct{}, len(removals))
	for _, s := range removals {
		negations[s] = struct{}{}
	}

	// Build a new slice containing only those items from "source"
	// that aren't in "negations".
	var result []string
	for _, s := range source {
		if _, found := negations[s]; !(found) {
			result = append(result, s)
		}
	}

	return result
}

// Valuer is the context return type relating to the [Telemetry] middleware. See the [Value] function for additional details.
type Valuer struct {
	// Headers retrieves a [http.Header] pointer representing [Telemetry] related headers.
	Headers http.Header `json:"headers"`

	// Path represents the request url's path component a part of its URI. This value is useful for telemetry-related implementations that
	// wish to provide additional information or context in spans for logging or event-related purposes.
	Path string `json:"path"`
}

// Options represents the configuration settings for the [Server] middleware component, including customizable server and header options.
type Options struct {
	// Headers includes telemetry-specific header(s) to store in a context key as derived from an http(s) request.
	//
	//	- The casings of these values are ignored.
	//
	// Default(s):
	//
	// 	- "portal"
	// 	- "device"
	// 	- "user"
	// 	- "travel"
	// 	- "traceparent"
	// 	- "tracestate"
	// 	- "x-cloud-trace-context"
	// 	- "sw8"
	// 	- "user-agent"
	// 	- "cookie"
	// 	- "authorization"
	// 	- "jwt"
	// 	- "x-request-id"
	// 	- "x-b3-traceid"
	// 	- "x-b3-spanid"
	// 	- "x-b3-parentspanid"
	// 	- "x-b3-sampled"
	// 	- "x-b3-flags"
	// 	- "x-ot-span-context"
	// 	- "x-api-version"
	// 	- "x-testing-authorization"
	// 	- "x-service-name"
	// 	- "x-service-version"
	//	- "x-server-name"
	//	- "x-amzn-trace-id"
	// 	- "x-amzn-parentspan-id"
	// 	- "x-amzn-sampled"
	// 	- "x-amzn-flags"
	// 	- "x-amzn-correlation-id"
	// 	- "x-amzn-trace-context"
	// 	- "x-amzn-parentspan-context"
	// 	- "x-amzn-sampled-context"
	// 	- "x-amzn-correlation-context"
	// 	- "x-amzn-trace-source"
	// 	- "x-amzn-parentspan-source"
	// 	- "x-amzn-sampled-source"
	// 	- "x-amzn-correlation-source"
	// 	- "x-amzn-date"
	// 	- "x-amzn-security-token"
	// 	- "x-amzn-cf-id"
	// 	- "x-amzn-cf-identity"
	Headers []string

	// Additions specifies additional headers to include with [Options.Headers]. Users looking to configure extra headers, without having to respecify the [Options.Headers] defaults,
	// are encouraged to use Extra.
	//
	//	- The casings of these values are ignored.
	Additions []string

	// Exclusions specifies any headers to exclude from both [Options.Headers] and [Options.Additions].
	//
	//	- The casings of these values are ignored.
	Exclusions []string

	// Debug enables log messages relating to identified [Telemetry] request headers. Defaults to false.
	Debug bool
}

// Telemetry represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Telemetry struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Server] middleware's [Options] and returns the updated middleware instance.
func (t *Telemetry) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if t.options == nil {
		t.options = &Options{
			Headers: []string{
				"portal",
				"device",
				"user",
				"travel",
				"traceparent",
				"tracestate",
				"x-cloud-trace-context",
				"sw8",
				"user-agent",
				"cookie",
				"authorization",
				"jwt",
				"x-request-id",
				"x-b3-traceid",
				"x-b3-spanid",
				"x-b3-parentspanid",
				"x-b3-sampled",
				"x-b3-flags",
				"x-ot-span-context",
				"x-api-version",
				"x-testing-authorization",
				"x-service-name",
				"x-service-version",
				"x-server-name",
				"x-amzn-trace-id",
				"x-amzn-parentspan-id",
				"x-amzn-sampled",
				"x-amzn-flags",
				"x-amzn-correlation-id",
				"x-amzn-trace-context",
				"x-amzn-parentspan-context",
				"x-amzn-sampled-context",
				"x-amzn-correlation-context",
				"x-amzn-trace-source",
				"x-amzn-parentspan-source",
				"x-amzn-sampled-source",
				"x-amzn-correlation-source",
				"x-amzn-date",
				"x-amzn-security-token",
				"x-amzn-cf-id",
				"x-amzn-cf-identity",
			},
			Additions:  []string{},
			Exclusions: []string{},
			Debug:      false,
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(t.options)
		}
	}

	return t
}

// Handler applies middleware settings to modify the request context and set response headers. It forwards the request to the next handler in the chain.
func (t *Telemetry) Handler(next http.Handler) http.Handler {
	t.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Merge the default headers + any additions.
		configuration := slices.Clone(merge(t.options.Headers, t.options.Additions))

		// Typecast all headers in the configuration slices to a more simple form.
		for index := 0; index < len(configuration); index++ {
			value := strings.ToLower(configuration[index])
			configuration[index] = value
		}

		// Typecast all headers in the exclusions array to a more simple form.
		exclusions := slices.Clone(t.options.Exclusions)
		for index := 0; index < len(exclusions); index++ {
			value := strings.ToLower(exclusions[index])
			exclusions[index] = value
		}

		// Remove all headers defined in exclusions from the configuration.
		configuration = remove(configuration, exclusions)

		// Establish the final headers that will be stored in context.
		headers := http.Header{}

		// Iterate through the list of the configuration headers, and then do a http.Header lookup (case-insensitive) for the key.
		for index := range configuration {
			header := configuration[index]

			k := http.CanonicalHeaderKey(header)
			v := slices.Clone(r.Header.Values(header))

			_, found := headers[k]
			if (found) || (v != nil && len(v) > 0) {
				for _, value := range v {
					headers.Add(k, value)
				}
			}
		}

		// Establish the final context valuer to be passed down the request.
		valuer := Valuer{
			Headers: headers,
			Path:    r.URL.Path,
		}

		// Cast the valuer context value to a pointer to provide additional information whether the middleware was enabled.
		ctx = context.WithValue(ctx, key, &valuer)

		// For unit-testing, the handler must only log, at most, once.
		if t.options.Debug {
			slog.DebugContext(ctx, "Telemetry Request Header(s)", slog.String("url", r.URL.String()), slog.Any("value", valuer))
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Telemetry] middleware, implementing [middleware.Configurable]. If [Telemetry.Settings] isn't called,
// then the [Telemetry.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Telemetry)
}

// Value retrieves a [Valuer] pointer representing [Telemetry] related [Valuer.Headers] and their associated [Valuer.Path]. If a nil value is returned, it can be
// assumed that the [Telemetry] middleware isn't enabled for the particular caller's chain. If the value has assigned an empty map to [Valuer.Headers],
// it's to be assumed the [Telemetry] middleware is enabled, however, no related, request header(s) were found.
func Value(ctx context.Context) (value *Valuer) {
	const t = "x-testing-key" // t represents a context key for unit-testing.

	if v, ok := ctx.Value(key).(*Valuer); ok {
		value = v
	} else if test, valid := ctx.Value(t).(*Valuer); valid {
		slog.Log(ctx, (slog.LevelDebug - 4), "Received Unit-Testing Context", slog.String("key", t))

		value = test
	} else {
		slog.WarnContext(ctx, "Unable to Typecast Context Key Value", slog.String("error", "Bad-Context-Evaluation"), slog.String("key", string(key)), slog.Any("value", ctx.Value(key)))
	}

	return
}

// Runtime assurance that [Telemetry] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Telemetry)(nil)
