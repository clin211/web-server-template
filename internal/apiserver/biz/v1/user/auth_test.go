package user

import (
	"context"
	"errors"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"reflect"
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
	biz.authorization = fakePermissions{values: []string{"user.manage"}}
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
	if !reflect.DeepEqual(response.GetPermissions(), []string{"user.manage"}) {
		t.Fatalf("permissions = %v", response.GetPermissions())
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

type fakePermissions struct {
	values []string
	err    error
}

func (s fakePermissions) Permissions(context.Context, string) ([]string, error) {
	return s.values, s.err
}

func TestInfoFailsWhenPermissionLookupFails(t *testing.T) {
	biz := New(&fakeStore{user: &fakeUserStore{get: func(context.Context, *where.Options) (*model.UserM, error) {
		return &model.UserM{UserID: "user-1"}, nil
	}}}, newTestTokenManager())
	biz.authorization = fakePermissions{err: errors.New("database unavailable")}
	if _, err := biz.Info(contextx.WithUserID(context.Background(), "user-1"), &v1.InfoRequest{}); !errors.Is(err, errno.ErrServiceUnavailable) {
		t.Fatalf("Info error = %v, want unavailable", err)
	}
}

func TestChangePasswordRejectsDifferentTargetBeforeLoadingUser(t *testing.T) {
	biz := New(&fakeStore{user: &fakeUserStore{}}, newTestTokenManager())
	_, err := biz.ChangePassword(contextx.WithUserID(context.Background(), "admin"), &v1.ChangePasswordRequest{UserID: "other", OldPassword: "abc123", NewPassword: "def123"})
	if !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatalf("ChangePassword error = %v, want permission denied", err)
	}
}
