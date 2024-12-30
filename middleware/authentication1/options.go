package authentication

import (
	"context"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"

	"verification-service/internal/library/middleware/types"
)

type Settings struct {
	Verification func(ctx context.Context, token string) (*jwt.Token, error) // Verification is a user-provided jwt-verification function.

	Level slog.Leveler // Level represents a [log/slog] log level - defaults to (slog.LevelDebug - 4) (trace)
}

type Variadic types.Variadic[Settings]

func settings() *Settings {
	return &Settings{
		Level: (slog.LevelDebug - 4),
	}
}
