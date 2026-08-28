package user

import (
	"context"
	"strings"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authn"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestInfoReturnsCurrentUserWithoutPassword(t *testing.T) {
	password, err := authn.Encrypt("strong-password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	userStore := &fakeUserStore{get: func(context.Context, *where.Options) (*model.UserM, error) {
		return &model.UserM{UserID: "user-1", Username: "alice", Password: password, Nickname: "Alice"}, nil
	}}
	biz := New(&fakeStore{user: userStore, outbox: &fakeOutboxStore{}}, newTestTokenManager())
	ctx := contextx.WithUserID(context.Background(), "user-1")

	response, err := biz.Info(ctx, &v1.InfoRequest{})
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if response.GetUser() == nil || response.GetUser().GetUserID() != "user-1" {
		t.Fatalf("Info() user = %#v, want current user", response.GetUser())
	}
	if response.GetUser().GetUsername() != "alice" {
		t.Fatalf("Info() user.username = %q, want alice", response.GetUser().GetUsername())
	}
	payload, err := protojson.Marshal(response.GetUser())
	if err != nil {
		t.Fatalf("marshal current user: %v", err)
	}
	if strings.Contains(string(payload), "password") {
		t.Fatal("Info() exposed password in current user")
	}
}

func TestLogoutRevokesCurrentTokenVersion(t *testing.T) {
	userStore := &fakeUserStore{}
	datastore := &fakeStore{user: userStore, outbox: &fakeOutboxStore{}}
	biz := New(datastore, newTestTokenManager())
	ctx := contextx.WithUserID(context.Background(), "user-1")

	if _, err := biz.Logout(ctx, &v1.LogoutRequest{}); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !datastore.txCalled {
		t.Fatal("Logout() did not use a transaction")
	}
	if userStore.updatedID != "user-1" {
		t.Fatalf("updated user ID = %q, want user-1", userStore.updatedID)
	}
	if _, ok := userStore.updates["token_version"]; !ok {
		t.Fatal("Logout() did not increment token_version")
	}
}
