package authentication

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/poly-gun/go-middleware"
)

// keyer is a private string type, unexported to ensure the context, constant key is always unique.
type keyer string

// key is the package's unexported context key. Only through the use of [Value] can the context's value be derived.
const key keyer = "versioning"

// Valuer is the context return type relating to the [Authentication] middleware. See the [Value] function for additional details.
type Valuer struct {
	Token *jwt.Token
}

// Options represents the configuration settings for the [Authentication] middleware component, including customizable server and header options.
type Options struct {
	Verification func(ctx context.Context, token string) (*jwt.Token, error) // Verification is a user-provided jwt-verification function.

	Level slog.Leveler // Level represents a [log/slog] log level - defaults to [slog.LevelDebug] - 4 (trace).
}

// Authentication represents a middleware component that applies configurable [Options] settings to HTTP requests. It
// embeds [middleware.Configurable] for [Options] configuration.
type Authentication struct {
	middleware.Configurable[Options]

	options *Options
}

// Settings applies configuration functions to modify the [Authentication] middleware's [Options] and returns the updated middleware instance.
func (a *Authentication) Settings(configuration ...func(o *Options)) middleware.Configurable[Options] {
	if a.options == nil {
		a.options = &Options{
			Level: (slog.LevelDebug - 4),
		}
	}

	for index := range configuration {
		if callable := configuration[index]; callable != nil {
			callable(a.options)
		}
	}

	return a
}

// Handler applies middleware settings to modify the request context and set response headers. It forwards the request to the next handler in the chain.
func (a *Authentication) Handler(next http.Handler) http.Handler {
	a.Settings() // Ensure the options field isn't nil.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var tokenstring string

		cookie, e := r.Cookie("token")
		if e == nil {
			tokenstring = cookie.Value
		} else {
			slog.Log(ctx, a.options.Level.Level(), "Cookie Not Found - Attempting Authorization Authentication")

			authorization := r.Header.Get("Authorization")
			if authorization == "" {
				authorization = r.Header.Get("X-Testing-Authorization") // To bypass proxy url header issues
			}

			if authorization != "" {
				partials := strings.Split(authorization, " ")
				slog.Log(ctx, a.options.Level.Level(), "Authorization Header Partial(s)", slog.Any("partials", partials))
				if len(partials) != 2 || partials[0] != "Bearer" {
					slog.WarnContext(ctx, "Invalid Authorization Format")
					http.Error(w, "Invalid Authorization Header Format", http.StatusUnauthorized)
					return
				}
			}

			if authorization == "" && errors.Is(e, http.ErrNoCookie) {
				slog.WarnContext(ctx, "No Valid Authorization Header or Cookie Found")
				http.Error(w, "Invalid JWT Token", http.StatusUnauthorized)
				return
			} else if authorization == "" {
				slog.WarnContext(ctx, "No Valid Authorization Header, and Unknown Cookie Error", slog.String("error", e.Error()))
				http.Error(w, "Invalid JWT Token", http.StatusUnauthorized)
				return
			}

			partials := strings.Split(authorization, " ")
			if len(partials) != 2 || partials[0] != "Bearer" {
				slog.WarnContext(ctx, "Invalid Authorization Format")
				http.Error(w, "Invalid Authorization Header Format", http.StatusUnauthorized)
				return
			}

			tokenstring = partials[1]
		}

		jwttoken, e := a.options.Verification(ctx, tokenstring)
		if e != nil {
			switch {
			case errors.Is(e, jwt.ErrTokenMalformed):
				const message = "Malformed JWT Token"

				slog.WarnContext(ctx, message)
				http.Error(w, message, http.StatusUnauthorized)
				return
			case errors.Is(e, jwt.ErrTokenSignatureInvalid):
				const message = "Invalid JWT Token Signature"

				slog.WarnContext(ctx, message)
				http.Error(w, message, http.StatusUnauthorized)
				return
			case errors.Is(e, jwt.ErrTokenExpired):
				const message = "Expired JWT Token"

				slog.WarnContext(ctx, message)
				http.Error(w, message, http.StatusUnauthorized)
				return
			case errors.Is(e, jwt.ErrTokenNotValidYet):
				const message = "Invalid Future JWT Token"

				slog.WarnContext(ctx, message)
				http.Error(w, message, http.StatusUnauthorized)
				return
			case errors.Is(e, jwt.ErrTokenInvalidAudience):
				const message = "Invalid Target Service (Audience)"

				slog.WarnContext(ctx, message)
				http.Error(w, message, http.StatusForbidden)
				return
			default:
				slog.ErrorContext(ctx, "Unhandled JWT Error", slog.String("error", e.Error()), slog.String("error-type", reflect.TypeOf(e).String()))
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		}

		slog.Log(ctx, a.options.Level.Level(), "JWT Token Structure", slog.Any("header(s)", jwttoken.Header), slog.Any("claim(s)", jwttoken.Claims))

		ctx = context.WithValue(ctx, key, &Valuer{
			Token: jwttoken,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// New creates a new instance of the [Authentication] middleware, implementing [middleware.Configurable]. If [Versions.Settings] isn't called,
// then the [Versions.Handler] function will hydrate the middleware's configuration with sane default(s) if applicable.
func New() middleware.Configurable[Options] {
	return new(Authentication)
}

// Value retrieves a [Valuer] pointer representing [Authentication] related context. If a nil value is returned, it can be
// assumed that the [Authentication] middleware isn't enabled for the particular caller's chain.
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

// Runtime assurance that [Authentication] satisfies [middleware.Configurable] requirement(s).
var _ middleware.Configurable[Options] = (*Authentication)(nil)
