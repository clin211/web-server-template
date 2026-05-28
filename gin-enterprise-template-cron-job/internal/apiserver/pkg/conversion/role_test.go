package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

func TestRoleModelToRoleV1(t *testing.T) {
	description := "管理员"
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700003600, 0)

	tests := []struct {
		name            string
		role            *model.RoleM
		wantRoleID      string
		wantRoleName    string
		wantRoleCode    string
		wantDescription string
		wantStatus      int32
		wantSortOrder   int32
		wantCreatedAt   int64
		wantUpdatedAt   int64
	}{
		{
			name:            "maps populated fields",
			role:            &model.RoleM{RoleID: "role-1", RoleName: "admin", RoleCode: "admin", Description: &description, Status: 1, SortOrder: 9, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantRoleID:      "role-1",
			wantRoleName:    "admin",
			wantRoleCode:    "admin",
			wantDescription: description,
			wantStatus:      1,
			wantSortOrder:   9,
			wantCreatedAt:   createdAt.Unix(),
			wantUpdatedAt:   updatedAt.Unix(),
		},
		{
			name:          "maps nil description to empty string",
			role:          &model.RoleM{RoleID: "role-2", RoleName: "user", RoleCode: "user", Status: 0, SortOrder: 3, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantRoleID:    "role-2",
			wantRoleName:  "user",
			wantRoleCode:  "user",
			wantStatus:    0,
			wantSortOrder: 3,
			wantCreatedAt: createdAt.Unix(),
			wantUpdatedAt: updatedAt.Unix(),
		},
		{
			name: "returns empty role for nil input",
			role: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleModelToRoleV1(tt.role)
			if got == nil {
				t.Fatal("RoleModelToRoleV1() returned nil")
			}
			if got.RoleID != tt.wantRoleID {
				t.Fatalf("RoleID = %q, want %q", got.RoleID, tt.wantRoleID)
			}
			if got.RoleName != tt.wantRoleName {
				t.Fatalf("RoleName = %q, want %q", got.RoleName, tt.wantRoleName)
			}
			if got.RoleCode != tt.wantRoleCode {
				t.Fatalf("RoleCode = %q, want %q", got.RoleCode, tt.wantRoleCode)
			}
			if got.Description != tt.wantDescription {
				t.Fatalf("Description = %q, want %q", got.Description, tt.wantDescription)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("Status = %d, want %d", got.Status, tt.wantStatus)
			}
			if got.SortOrder != tt.wantSortOrder {
				t.Fatalf("SortOrder = %d, want %d", got.SortOrder, tt.wantSortOrder)
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

func TestRoleModelListToRoleV1List(t *testing.T) {
	roles := []*model.RoleM{
		{RoleID: "role-1", RoleName: "admin", RoleCode: "admin"},
		{RoleID: "role-2", RoleName: "user", RoleCode: "user"},
	}

	got := RoleModelListToRoleV1List(roles)
	if len(got) != len(roles) {
		t.Fatalf("len(RoleModelListToRoleV1List()) = %d, want %d", len(got), len(roles))
	}
	for i, role := range roles {
		if got[i].RoleID != role.RoleID {
			t.Fatalf("Role[%d].RoleID = %q, want %q", i, got[i].RoleID, role.RoleID)
		}
	}
}
