package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

func TestPermissionModelToPermissionV1(t *testing.T) {
	resourcePath := "/system/user/list"
	description := "用户列表权限"
	parentID := "perm-parent"
	path := "system.user.list"
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700003600, 0)

	tests := []struct {
		name             string
		permission       *model.PermissionM
		wantPermissionID string
		wantName         string
		wantCode         string
		wantResourceType string
		wantResourcePath string
		wantAction       string
		wantDescription  string
		wantParentID     string
		wantPath         string
		wantStatus       int32
		wantCreatedAt    int64
		wantUpdatedAt    int64
	}{
		{
			name:             "maps populated fields",
			permission:       &model.PermissionM{PermissionID: "perm-1", PermissionName: "用户列表", PermissionCode: "user:list", ResourceType: "button", ResourcePath: &resourcePath, Action: "GET", Description: &description, ParentID: &parentID, Path: &path, Status: 1, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantPermissionID: "perm-1",
			wantName:         "用户列表",
			wantCode:         "user:list",
			wantResourceType: "button",
			wantResourcePath: resourcePath,
			wantAction:       "GET",
			wantDescription:  description,
			wantParentID:     parentID,
			wantPath:         path,
			wantStatus:       1,
			wantCreatedAt:    createdAt.Unix(),
			wantUpdatedAt:    updatedAt.Unix(),
		},
		{
			name:             "maps nil pointers to empty strings",
			permission:       &model.PermissionM{PermissionID: "perm-2", PermissionName: "角色管理", PermissionCode: "role:list", ResourceType: "menu", Action: "GET", Status: 0, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantPermissionID: "perm-2",
			wantName:         "角色管理",
			wantCode:         "role:list",
			wantResourceType: "menu",
			wantAction:       "GET",
			wantStatus:       0,
			wantCreatedAt:    createdAt.Unix(),
			wantUpdatedAt:    updatedAt.Unix(),
		},
		{
			name:       "returns empty permission for nil input",
			permission: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PermissionModelToPermissionV1(tt.permission)
			if got == nil {
				t.Fatal("PermissionModelToPermissionV1() returned nil")
			}
			if got.PermissionID != tt.wantPermissionID {
				t.Fatalf("PermissionID = %q, want %q", got.PermissionID, tt.wantPermissionID)
			}
			if got.PermissionName != tt.wantName {
				t.Fatalf("PermissionName = %q, want %q", got.PermissionName, tt.wantName)
			}
			if got.PermissionCode != tt.wantCode {
				t.Fatalf("PermissionCode = %q, want %q", got.PermissionCode, tt.wantCode)
			}
			if got.ResourceType != tt.wantResourceType {
				t.Fatalf("ResourceType = %q, want %q", got.ResourceType, tt.wantResourceType)
			}
			if got.ResourcePath != tt.wantResourcePath {
				t.Fatalf("ResourcePath = %q, want %q", got.ResourcePath, tt.wantResourcePath)
			}
			if got.Action != tt.wantAction {
				t.Fatalf("Action = %q, want %q", got.Action, tt.wantAction)
			}
			if got.Description != tt.wantDescription {
				t.Fatalf("Description = %q, want %q", got.Description, tt.wantDescription)
			}
			if got.ParentID != tt.wantParentID {
				t.Fatalf("ParentID = %q, want %q", got.ParentID, tt.wantParentID)
			}
			if got.Path != tt.wantPath {
				t.Fatalf("Path = %q, want %q", got.Path, tt.wantPath)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("Status = %d, want %d", got.Status, tt.wantStatus)
			}
			if got.CreatedAt != tt.wantCreatedAt {
				t.Fatalf("CreatedAt = %d, want %d", got.CreatedAt, tt.wantCreatedAt)
			}
			if got.UpdatedAt != tt.wantUpdatedAt {
				t.Fatalf("UpdatedAt = %d, want %d", got.UpdatedAt, tt.wantUpdatedAt)
			}
		})
	}
}

func TestPermissionModelListToPermissionTreeV1(t *testing.T) {
	rootID := "perm-root"
	childParent := rootID
	path := "/system"
	permissions := []*model.PermissionM{
		{PermissionID: rootID, PermissionName: "系统管理", PermissionCode: "system", ResourceType: "menu", ResourcePath: &path},
		{PermissionID: "perm-child", PermissionName: "用户列表", PermissionCode: "user:list", ResourceType: "button", ParentID: &childParent},
	}

	got := PermissionModelListToPermissionTreeV1(permissions, map[string]bool{"perm-child": true})
	if len(got) != 1 {
		t.Fatalf("len(PermissionModelListToPermissionTreeV1()) = %d, want 1", len(got))
	}
	if got[0].PermissionID != rootID {
		t.Fatalf("root PermissionID = %q, want %q", got[0].PermissionID, rootID)
	}
	if len(got[0].Children) != 1 {
		t.Fatalf("len(root.Children) = %d, want 1", len(got[0].Children))
	}
	child := got[0].Children[0]
	if child.PermissionID != "perm-child" {
		t.Fatalf("child PermissionID = %q, want %q", child.PermissionID, "perm-child")
	}
	if !child.Assigned {
		t.Fatal("child.Assigned = false, want true")
	}
}

func TestPermissionModelToPermissionTreeNodeV1(t *testing.T) {
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700003600, 0)

	tests := []struct {
		name    string
		perm    *model.PermissionM
		wantID  string
		wantNil bool
	}{
		{
			name:    "maps populated fields",
			perm:    &model.PermissionM{PermissionID: "perm-1", PermissionName: "测试权限", PermissionCode: "test:read", ResourceType: "button", Action: "GET", Status: 0, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantID:  "perm-1",
			wantNil: false,
		},
		{
			name:    "returns empty node for nil input",
			perm:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PermissionModelToPermissionTreeNodeV1(tt.perm)
			if tt.wantNil {
				if got.Permission != nil {
					t.Fatalf("Permission = %v, want nil", got.Permission)
				}
				return
			}
			if got.Permission == nil {
				t.Fatal("Permission = nil, want non-nil")
			}
			if got.Permission.PermissionID != tt.wantID {
				t.Fatalf("Permission.PermissionID = %q, want %q", got.Permission.PermissionID, tt.wantID)
			}
			if got.Permission.PermissionName != tt.perm.PermissionName {
				t.Fatalf("Permission.PermissionName = %q, want %q", got.Permission.PermissionName, tt.perm.PermissionName)
			}
		})
	}
}

func TestBuildPermissionTree(t *testing.T) {
	// 这个测试验证树形结构的构建逻辑
	// 注意：实际的 buildPermissionTree 函数在 biz 层，这里只测试转换层逻辑

	rootID := "perm-root"
	childParent := rootID

	permissions := []*model.PermissionM{
		{PermissionID: rootID, PermissionName: "根权限", PermissionCode: "root", ResourceType: "menu", Action: "MENU"},
		{PermissionID: "perm-child-1", PermissionName: "子权限1", PermissionCode: "child:1", ResourceType: "button", Action: "GET", ParentID: &childParent},
		{PermissionID: "perm-child-2", PermissionName: "子权限2", PermissionCode: "child:2", ResourceType: "button", Action: "POST", ParentID: &childParent},
	}

	// 验证 PermissionTreeNodeFields 能正确处理树节点
	for _, perm := range permissions {
		node := PermissionTreeNodeFields(perm)
		if node.Permission.PermissionID != perm.PermissionID {
			t.Errorf("PermissionID mismatch for %s: got %s, want %s",
				perm.PermissionName, node.Permission.PermissionID, perm.PermissionID)
		}
	}
}
