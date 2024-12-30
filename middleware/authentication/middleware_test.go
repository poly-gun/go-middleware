package authentication_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/poly-gun/go-middleware/middleware/authentication"
)

func Test(t *testing.T) {
	var verify = func(ctx context.Context, t string) (*jwt.Token, error) {
		token, e := jwt.Parse(t, func(token *jwt.Token) (interface{}, error) {
			method, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}

			_ = method

			return []byte("mHTuL3Xko1FKxqxEa3WFrVXyfQEOsfsODyusTDgD9F4"), nil
		})

		if e != nil {
			slog.WarnContext(ctx, "Error Parsing JWT Token", slog.String("error", e.Error()), slog.String("jwt", t))
			return nil, e
		}

		switch {
		case token.Valid:
			slog.DebugContext(ctx, "Basic Token Parsing was Successful - Vetting Additional Claims")

			return token, nil
		case errors.Is(e, jwt.ErrTokenMalformed):
			slog.WarnContext(ctx, "Unable to Verify Malformed String as JWT Token", slog.String("error", e.Error()))
		case errors.Is(e, jwt.ErrTokenSignatureInvalid):
			slog.WarnContext(ctx, "Invalid JWT Signature", slog.String("error", e.Error()))
		case errors.Is(e, jwt.ErrTokenExpired):
			slog.WarnContext(ctx, "Expired JWT Token", slog.String("error", e.Error()))
		case errors.Is(e, jwt.ErrTokenNotValidYet):
			slog.WarnContext(ctx, "Received a Future, Valid JWT Token", slog.String("error", e.Error()))
		default:
			slog.ErrorContext(ctx, "Unknown Error While Attempting to Validate JWT Token", slog.String("error", e.Error()))
		}

		return nil, e
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		datum := map[string]interface{}{
			"Key-1": "Value-1",
			"Key-2": "Value-2",
			"Key-3": "Value-3",
		}

		defer json.NewEncoder(w).Encode(datum)

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		return
	})

	t.Run("Middleware", func(t *testing.T) {
		t.Run("Successful-Unauthorized-Response", func(t *testing.T) {
			server := httptest.NewServer(authentication.New().Settings(func(o *authentication.Options) { o.Verification = verify }).Handler(handler))

			defer server.Close()

			client := server.Client()
			request, e := http.NewRequest(http.MethodGet, server.URL, nil)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Request: %v", e)
			}

			response, e := client.Do(request)
			if e != nil {
				t.Fatalf("Unexpected Error While Generating Response: %v", e)
			}

			defer response.Body.Close()

			if response.StatusCode != http.StatusUnauthorized {
				t.Logf("Expected Unauthorized Status-Code")
			}
		})
	})

	t.Run("Context", func(t *testing.T) {
		t.Run("Default", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			value := authentication.Value(ctx)

			if value != nil {
				t.Errorf("Unexpected Non-Default Value: %v", value)
			} else {
				t.Logf("Successful Default Value Received = %v", value)
			}
		})
	})

	t.Run("Logging", func(t *testing.T) {
		t.Run("Context-Key-Value-Warning-Log-Level", func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{
				AddSource:   true,
				Level:       slog.LevelDebug,
				ReplaceAttr: nil,
			}))

			slog.SetDefault(logger)

			ctx := context.Background()

			authentication.Value(ctx)

			var message map[string]interface{}
			if e := json.Unmarshal(buffer.Bytes(), &message); e != nil {
				t.Fatalf("Fatal, Unexpected Error While Unmarshalling Log Message: %v", e)
			}

			if v, ok := message["level"]; ok {
				if typecast, valid := v.(string); valid {
					if typecast == slog.LevelWarn.String() {
						t.Logf("Successful, Expected Log-Level Level Achieved")
					} else {
						t.Errorf("Unexpected Log-Level Level: %s", typecast)
					}
				} else {
					t.Errorf("Unable to Typecast Level to String Type: %v", v)
				}
			} else {
				t.Errorf("No Valid Level Key Found: %v", message)
			}
		})
	})
}
