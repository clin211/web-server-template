package gin

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ginframework "github.com/gin-gonic/gin"
)

func TestObservabilityDoesNotLogBodies(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)

	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "successful request", statusCode: http.StatusOK},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			previous := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
			t.Cleanup(func() { slog.SetDefault(previous) })

			engine := ginframework.New()
			engine.Use(Observability())
			engine.POST("/v1/users/:userID", func(c *ginframework.Context) {
				c.JSON(tt.statusCode, ginframework.H{"token": "response-secret"})
			})

			request := httptest.NewRequest(http.MethodPost, "/v1/users/user-1", strings.NewReader(`{"password":"request-secret"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)

			logs := output.String()
			for _, secret := range []string{"request-secret", "response-secret", "password", "token"} {
				if strings.Contains(logs, secret) {
					t.Errorf("logs contain sensitive body value %q: %s", secret, logs)
				}
			}
			if !strings.Contains(logs, `"http.route":"/v1/users/:userID"`) {
				t.Errorf("logs do not contain route template: %s", logs)
			}
		})
	}
}
