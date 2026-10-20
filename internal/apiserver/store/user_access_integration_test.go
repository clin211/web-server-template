package store_test

import (
	"context"
	"errors"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"slices"
	"testing"

	userbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/user"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/validation"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func TestIntegrationUserEffectivePermissionsAndCreation(t *testing.T) {
	db, core := testDatabase(t)
	ctx := context.Background()
	authorization := store.NewAuthorizationStore(core)
	for _, id := range []string{"owner", "missing"} {
		permissions, err := authorization.Permissions(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(permissions, "user.manage") {
			t.Fatalf("unprivileged %s has management grant", id)
		}
	}
	if err := db.Exec("INSERT INTO user_role VALUES('owner','role_admin')").Error; err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		"INSERT INTO role(role_id,name) VALUES('duplicate','duplicate')",
		"INSERT INTO role_permission VALUES('duplicate','user.manage')",
		"INSERT INTO user_role VALUES('owner','duplicate')",
	} {
		if err := db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	permissions, err := authorization.Permissions(ctx, "owner")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, permission := range permissions {
		if permission == "user.manage" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("duplicate effective grant count = %d", count)
	}
	validator := validation.New(core)
	request := &v1.CreateUserRequest{Username: "newmember", Password: "abc123", Email: "new@example.com", Phone: "13800138000"}
	for _, id := range []string{"", "other"} {
		err := validator.ValidateCreateUserRequest(contextx.WithUserID(ctx, id), request)
		if !errors.Is(err, errno.ErrPermissionDenied) {
			t.Fatalf("create as %q = %v", id, err)
		}
	}
	admin := contextx.WithUserID(ctx, "owner")
	if err := validator.ValidateCreateUserRequest(admin, request); err != nil {
		t.Fatal(err)
	}
	biz := userbiz.New(core, nil)
	created, err := biz.Create(admin, request)
	if err != nil {
		t.Fatal(err)
	}
	total, first, err := core.User().List(ctx, where.NewWhere(where.WithLimit(2)))
	if err != nil || len(first) != 2 || total != 3 {
		t.Fatalf("first page total=%d users=%v err=%v", total, first, err)
	}
	opts := where.NewWhere(where.WithLimit(2), where.WithCursor(first[1].ID))
	secondTotal, second, err := core.User().List(ctx, opts)
	if err != nil || secondTotal != total || len(second) != 1 || second[0].UserID == first[0].UserID || second[0].UserID == first[1].UserID || opts.Cursor == nil || opts.Limit != 2 {
		t.Fatalf("second page total=%d users=%v err=%v", secondTotal, second, err)
	}
	grants, err := authorization.Permissions(ctx, created.UserID)
	if err != nil || slices.Contains(grants, "user.manage") {
		t.Fatalf("created account grants = %v, err = %v", grants, err)
	}
	info, err := biz.Info(admin, &v1.InfoRequest{})
	if err != nil || !slices.Contains(info.GetPermissions(), "user.manage") {
		t.Fatalf("Info grants = %v, err = %v", info, err)
	}
	if err := validator.ValidateChangePasswordRequest(admin, &v1.ChangePasswordRequest{UserID: "other", OldPassword: "abc123", NewPassword: "def123"}); !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatalf("admin changing other password = %v", err)
	}
	if err := db.Exec(`UPDATE "user" SET status=1 WHERE user_id='owner'`).Error; err != nil {
		t.Fatal(err)
	}
	grants, err = authorization.Permissions(ctx, "owner")
	if err != nil || len(grants) != 0 {
		t.Fatalf("disabled grants = %v, err = %v", grants, err)
	}
	if err := validator.ValidateCreateUserRequest(admin, request); !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatalf("disabled create = %v", err)
	}
}
