package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

func TestMenuModelToMenuV1(t *testing.T) {
	parentID := "menu-parent"
	icon := "user"
	path := "/system/user"
	component := "system/user/index"
	permissionID := "perm-1"
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700003600, 0)

	tests := []struct {
		name             string
		menu             *model.MenuM
		wantMenuID       string
		wantParentID     string
		wantMenuName     string
		wantMenuCode     string
		wantMenuType     string
		wantIcon         string
		wantPath         string
		wantComponent    string
		wantPermissionID string
		wantSortOrder    int32
		wantVisible      int32
		wantStatus       int32
		wantCreatedAt    int64
		wantUpdatedAt    int64
	}{
		{
			name:             "maps populated fields",
			menu:             &model.MenuM{MenuID: "menu-1", ParentID: &parentID, MenuName: "用户管理", MenuCode: "user", MenuType: "page", Icon: &icon, Path: &path, Component: &component, PermissionID: &permissionID, SortOrder: 10, Visible: 1, Status: 0, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantMenuID:       "menu-1",
			wantParentID:     parentID,
			wantMenuName:     "用户管理",
			wantMenuCode:     "user",
			wantMenuType:     "page",
			wantIcon:         icon,
			wantPath:         path,
			wantComponent:    component,
			wantPermissionID: permissionID,
			wantSortOrder:    10,
			wantVisible:      1,
			wantStatus:       0,
			wantCreatedAt:    createdAt.Unix(),
			wantUpdatedAt:    updatedAt.Unix(),
		},
		{
			name:          "maps nil pointers to empty strings",
			menu:          &model.MenuM{MenuID: "menu-2", MenuName: "系统管理", MenuCode: "system", MenuType: "menu", SortOrder: 1, Visible: 0, Status: 1, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantMenuID:    "menu-2",
			wantMenuName:  "系统管理",
			wantMenuCode:  "system",
			wantMenuType:  "menu",
			wantSortOrder: 1,
			wantVisible:   0,
			wantStatus:    1,
			wantCreatedAt: createdAt.Unix(),
			wantUpdatedAt: updatedAt.Unix(),
		},
		{
			name: "returns empty menu for nil input",
			menu: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MenuModelToMenuV1(tt.menu)
			if got == nil {
				t.Fatal("MenuModelToMenuV1() returned nil")
			}
			if got.MenuID != tt.wantMenuID {
				t.Fatalf("MenuID = %q, want %q", got.MenuID, tt.wantMenuID)
			}
			if got.ParentID != tt.wantParentID {
				t.Fatalf("ParentID = %q, want %q", got.ParentID, tt.wantParentID)
			}
			if got.MenuName != tt.wantMenuName {
				t.Fatalf("MenuName = %q, want %q", got.MenuName, tt.wantMenuName)
			}
			if got.MenuCode != tt.wantMenuCode {
				t.Fatalf("MenuCode = %q, want %q", got.MenuCode, tt.wantMenuCode)
			}
			if got.MenuType != tt.wantMenuType {
				t.Fatalf("MenuType = %q, want %q", got.MenuType, tt.wantMenuType)
			}
			if derefString(got.Icon) != tt.wantIcon {
				t.Fatalf("Icon = %q, want %q", derefString(got.Icon), tt.wantIcon)
			}
			if derefString(got.Path) != tt.wantPath {
				t.Fatalf("Path = %q, want %q", derefString(got.Path), tt.wantPath)
			}
			if derefString(got.Component) != tt.wantComponent {
				t.Fatalf("Component = %q, want %q", derefString(got.Component), tt.wantComponent)
			}
			if derefString(got.PermissionID) != tt.wantPermissionID {
				t.Fatalf("PermissionID = %q, want %q", derefString(got.PermissionID), tt.wantPermissionID)
			}
			if got.SortOrder != tt.wantSortOrder {
				t.Fatalf("SortOrder = %d, want %d", got.SortOrder, tt.wantSortOrder)
			}
			if got.Visible != tt.wantVisible {
				t.Fatalf("Visible = %d, want %d", got.Visible, tt.wantVisible)
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

func TestMenuModelListToMenuTreeV1(t *testing.T) {
	rootID := "menu-root"
	childParentID := rootID
	menus := []*model.MenuM{
		{MenuID: rootID, MenuName: "系统管理", MenuCode: "system", MenuType: "menu"},
		{MenuID: "menu-child", ParentID: &childParentID, MenuName: "用户管理", MenuCode: "user", MenuType: "page"},
	}

	got := MenuModelListToMenuTreeV1(menus)
	if len(got) != 1 {
		t.Fatalf("len(MenuModelListToMenuTreeV1()) = %d, want 1", len(got))
	}
	if got[0].MenuID != rootID {
		t.Fatalf("root MenuID = %q, want %q", got[0].MenuID, rootID)
	}
	if len(got[0].Children) != 1 {
		t.Fatalf("len(root.Children) = %d, want 1", len(got[0].Children))
	}
	child := got[0].Children[0]
	if child.MenuID != "menu-child" {
		t.Fatalf("child MenuID = %q, want %q", child.MenuID, "menu-child")
	}
	if child.ParentID != rootID {
		t.Fatalf("child ParentID = %q, want %q", child.ParentID, rootID)
	}
}
