package authentication

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"verification-service/internal/library/middleware/types"
)

type generic struct {
	types.Valuer[*Authentication]

	options *Settings
}

func (g *generic) Configuration(options ...Variadic) Implementation {
	var o = settings()
	for _, option := range options {
		option(o)
	}

	g.options = o

	return g
}

func (*generic) Value(ctx context.Context) *Authentication {
	v, ok := ctx.Value(key).(*Authentication)

	if !ok {
		slog.WarnContext(ctx, fmt.Sprintf("Authentication Not Found in CTX"))
		return nil
	}

	return v
}

func (g *generic) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		authentication := &Authentication{}

		var tokenstring string

		cookie, e := r.Cookie("token")
		if e == nil {
			tokenstring = cookie.Value
		} else {
			slog.Log(ctx, g.options.Level.Level(), "Cookie Not Found - Attempting Authorization Authentication")

			authorization := r.Header.Get("Authorization")
			if authorization == "" {
				authorization = r.Header.Get("X-Testing-Authorization") // To bypass proxy url header issues
			}

			if authorization != "" {
				partials := strings.Split(authorization, " ")
				slog.Log(ctx, g.options.Level.Level(), "Authorization Header Partial(s)", slog.Any("partials", partials))
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

		jwttoken, e := g.options.Verification(ctx, tokenstring)
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
			default:
				slog.ErrorContext(ctx, "Unhandled JWT Error", slog.String("error", e.Error()))
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		}

		slog.Log(ctx, g.options.Level.Level(), "JWT Token Structure", slog.Any("header(s)", jwttoken.Header), slog.Any("claim(s)", jwttoken.Claims))

		{ // --> token
			value := jwttoken

			authentication.Token = value
		}

		ctx = context.WithValue(ctx, key, authentication)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
