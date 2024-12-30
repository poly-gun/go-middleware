package authentication

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type Authentication struct {
	Token *jwt.Token
}

type Implementation interface {
	Value(ctx context.Context) *Authentication
	Configuration(options ...Variadic) Implementation
	Middleware(next http.Handler) http.Handler
}

func New() Implementation {
	return &generic{
		options: settings(),
	}
}
