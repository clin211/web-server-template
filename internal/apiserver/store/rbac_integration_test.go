package store_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
)

func TestIntegrationRBACReadbackAndProtections(t *testing.T) {
	db, core := testDatabase(t)
	if err := db.Exec("INSERT INTO role(role_id,name) VALUES('role_manager','manager')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO role_permission VALUES('role_manager','rbac.manage')").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO user_role VALUES('owner','role_manager')").Error; err != nil {
		t.Fatal(err)
	}
	service := taskbiz.New(core)
	manager := contextx.WithUserID(context.Background(), "owner")
	ordinary := contextx.WithUserID(context.Background(), "other")
	if err := service.Require(manager, "user.manage", ""); !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatalf("manager unexpectedly has user.manage: %v", err)
	}
	if _, err := service.Role(ordinary, "role_manager"); !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatalf("ordinary role read: %v", err)
	}
	if err := service.RoleMutation(manager, "create", "role_editor", "编辑人员", nil); err != nil {
		t.Fatal(err)
	}
	if err := service.RoleMutation(manager, "permissions", "role_editor", "", []string{"task.view", "user.manage"}); err != nil {
		t.Fatal(err)
	}
	role, err := service.Role(manager, "role_editor")
	if err != nil || !reflect.DeepEqual(role.Permissions, []string{"task.view", "user.manage"}) {
		t.Fatalf("read role = %+v, %v", role, err)
	}
	if err := service.RoleMutation(manager, "assign", "role_editor", "other", nil); err != nil {
		t.Fatal(err)
	}
	grants, err := service.UserRoles(manager, "other")
	if err != nil || !reflect.DeepEqual(grants.Permissions, []string{"result.download", "schedule.manage", "task.cancel", "task.retry", "task.submit", "task.view", "user.manage"}) {
		t.Fatalf("user grants = %+v, %v", grants, err)
	}
	if err := service.RoleMutation(manager, "permissions", "role_editor", "", []string{"missing.permission"}); err == nil {
		t.Fatal("unknown permission accepted")
	}
	role, err = service.Role(manager, "role_editor")
	if err != nil || !reflect.DeepEqual(role.Permissions, []string{"task.view", "user.manage"}) {
		t.Fatalf("failed write lost old grants: %+v, %v", role, err)
	}
	if err := service.RoleMutation(manager, "permissions", "role_user", "", nil); err == nil {
		t.Fatal("builtin role editable")
	}
	if err := service.RoleMutation(manager, "revoke", "role_editor", "other", nil); err != nil {
		t.Fatal(err)
	}
	grants, err = service.UserRoles(manager, "other")
	if err != nil || len(grants.Roles) != 1 || grants.Roles[0].RoleID != "role_user" {
		t.Fatalf("revoke readback = %+v, %v", grants, err)
	}
	if _, err := service.Role(manager, "missing"); !errors.Is(err, errno.ErrNotFound) {
		t.Fatalf("missing role = %v", err)
	}
	if _, err := service.UserRoles(manager, "missing"); !errors.Is(err, errno.ErrNotFound) {
		t.Fatalf("missing user = %v", err)
	}
	if err := service.RoleMutation(manager, "assign", "role_admin", "owner", nil); err != nil {
		t.Fatal(err)
	}
	if err := service.RoleMutation(manager, "revoke", "role_admin", "owner", nil); err == nil {
		t.Fatal("last administrator revoked")
	}
	if err := service.RoleMutation(manager, "create", "role_empty", "  ", nil); !errors.Is(err, errno.ErrInvalidArgument) {
		t.Fatalf("blank role name = %v", err)
	}
}
