package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/gin-gonic/gin"
)

func TestRBACReadRoutesRequireAuthentication(t *testing.T) {
	router := gin.New()
	// Existing user routes and the RBAC subresource must share the same parameter.
	router.GET("/v1/users/:userID", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	installRBACRoutes(router.Group("/v1"), taskbiz.New(nil))
	for _, path := range []string{"/v1/roles/role_admin", "/v1/users/member/roles"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body struct {
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Reason != "Auth.Unauthenticated" {
				t.Fatalf("reason = %q", body.Reason)
			}
		})
	}
}
