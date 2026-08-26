package gin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/clin211/gin-enterprise-template/pkg/token"
)

const authnTestKey = "test-authentication-key-with-sufficient-length"

type fakeUserRetriever struct {
	user *model.UserM
	err  error
}

func (r *fakeUserRetriever) GetUser(context.Context, string) (*model.UserM, error) {
	return r.user, r.err
}

func TestAuthnMiddlewareReturnsStableErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name      string
		header    string
		retriever *fakeUserRetriever
		wantCode  errorsx.BizCode
		wantMsg   string
	}{
		{
			name:      "malformed token",
			header:    "Bearer not-a-jwt",
			retriever: &fakeUserRetriever{},
			wantCode:  errorsx.CodeAuthTokenInvalid,
			wantMsg:   errno.ErrTokenInvalid.Message,
		},
		{
			name:      "user lookup failure",
			header:    signedAccessToken(t, "user-1"),
			retriever: &fakeUserRetriever{err: errors.New("postgres connection string leaked")},
			wantCode:  errorsx.CodeAuthUnauthenticated,
			wantMsg:   errno.ErrUnauthenticated.Message,
		},
		{
			name:      "retrieved identity does not match token",
			header:    signedAccessToken(t, "user-1"),
			retriever: &fakeUserRetriever{user: &model.UserM{UserID: "user-2"}},
			wantCode:  errorsx.CodeAuthUnauthenticated,
			wantMsg:   errno.ErrUnauthenticated.Message,
		},
		{
			name:      "disabled user",
			header:    signedAccessToken(t, "user-1"),
			retriever: &fakeUserRetriever{user: &model.UserM{UserID: "user-1", Status: 1}},
			wantCode:  errorsx.CodeUserPermissionDenied,
			wantMsg:   errno.ErrUserDisabled.Message,
		},
		{
			name:      "token version is stale",
			header:    signedAccessToken(t, "user-1"),
			retriever: &fakeUserRetriever{user: &model.UserM{UserID: "user-1", TokenVersion: 1}},
			wantCode:  errorsx.CodeAuthUnauthenticated,
			wantMsg:   errno.ErrUnauthenticated.Message,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodGet, "/v1/users/user-1", nil)
			req.Header.Set("Authorization", tt.header)
			c.Request = req

			AuthnMiddleware(tt.retriever, authnTestTokenManager())(c)

			if !c.IsAborted() {
				t.Fatal("middleware did not abort the request")
			}
			var response errorsx.APIResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != int(tt.wantCode) || response.Message != tt.wantMsg {
				t.Fatalf("response = %+v, want code=%d message=%q", response, tt.wantCode, tt.wantMsg)
			}
		})
	}
}

func TestAuthnMiddlewareAddsActiveUserToContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/v1/users/user-1", nil)
	req.Header.Set("Authorization", signedAccessToken(t, "user-1"))
	c.Request = req

	AuthnMiddleware(&fakeUserRetriever{user: &model.UserM{
		UserID:   "user-1",
		Username: "alice",
		Status:   0,
	}}, authnTestTokenManager())(c)

	if c.IsAborted() {
		t.Fatal("middleware unexpectedly aborted the request")
	}
	if got := contextx.UserID(c.Request.Context()); got != "user-1" {
		t.Fatalf("context userID = %q, want user-1", got)
	}
	if got := contextx.Username(c.Request.Context()); got != "alice" {
		t.Fatalf("context username = %q, want alice", got)
	}
}

func TestRefreshAuthnMiddlewareRejectsDisabledUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPut, "/v1/auth/refresh-token", nil)
	req.Header.Set("Authorization", "Bearer "+signedRefreshToken(t, "user-1"))
	c.Request = req

	RefreshAuthnMiddleware(&fakeUserRetriever{user: &model.UserM{UserID: "user-1", Status: 1}}, authnTestTokenManager())(c)

	if !c.IsAborted() {
		t.Fatal("middleware did not abort the request")
	}
	var response errorsx.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Message != errno.ErrUserDisabled.Message {
		t.Fatalf("response message = %q, want %q", response.Message, errno.ErrUserDisabled.Message)
	}
}

func signedAccessToken(t *testing.T, userID string) string {
	t.Helper()
	accessToken, _, _, _, err := authnTestTokenManager().SignWithVersion(userID, 0)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	return "Bearer " + accessToken
}

func signedRefreshToken(t *testing.T, userID string) string {
	t.Helper()
	_, refreshToken, _, _, err := authnTestTokenManager().SignWithVersion(userID, 0)
	if err != nil {
		t.Fatalf("sign refresh token: %v", err)
	}
	return refreshToken
}

func authnTestTokenManager() *token.Manager {
	return token.NewManager(authnTestKey, time.Hour, 24*time.Hour, token.WithIdentityKey(known.XUserID))
}
