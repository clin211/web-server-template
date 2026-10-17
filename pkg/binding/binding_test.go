package binding

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMultiSourceBindingPreservesGinValidation(t *testing.T) {
	t.Run("ordinary Gin binding still validates", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
		var input struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err == nil {
			t.Fatal("importing multi-source binding disabled ordinary Gin validation")
		}
	})
	t.Run("validate after all sources are bound", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Params = gin.Params{{Key: "id", Value: "user-1"}}
		c.Request = httptest.NewRequest("POST", "/users/user-1?filter=active", strings.NewReader(`{"name":"Alice"}`))
		var input struct {
			ID     string `uri:"id" binding:"required"`
			Name   string `json:"name" binding:"required"`
			Filter string `form:"filter" binding:"required"`
		}
		if err := Bind(c, &input, URI, JSON, Query); err != nil {
			t.Fatal(err)
		}
		if input.ID != "user-1" || input.Name != "Alice" || input.Filter != "active" {
			t.Fatalf("bound input=%+v", input)
		}
	})
}
