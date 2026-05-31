package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	menuv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/menu"
	permissionv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/permission"
	rolev1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/role"
	scheduledtaskv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/scheduled_task"
	userv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/user"
	userrolev1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/user_role"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/validation"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateUserBindsPathUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		pathUserID   string
		body         string
		wantUserID   string
		wantNickname string
	}{
		{
			name:         "binds path userID when body omits it",
			pathUserID:   "path-user-id",
			body:         `{"nickname":"neo"}`,
			wantUserID:   "path-user-id",
			wantNickname: "neo",
		},
		{
			name:         "uri overrides body userID",
			pathUserID:   "path-user-id",
			body:         `{"userID":"body-user-id","nickname":"trinity"}`,
			wantUserID:   "path-user-id",
			wantNickname: "trinity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userBiz := &stubUserBiz{}
			h := NewHandler(stubBiz{userBiz: userBiz}, &validation.Validator{})

			r := gin.New()
			r.Use(func(c *gin.Context) {
				ctx := contextx.WithUsername(c.Request.Context(), known.AdminUsername)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			r.PUT("/v1/users/:userID", h.UpdateUser)

			req := httptest.NewRequest(http.MethodPut, "/v1/users/"+tt.pathUserID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.NotNil(t, userBiz.lastUpdateRequest)
			assert.Equal(t, tt.wantUserID, userBiz.lastUpdateRequest.GetUserID())
			require.NotNil(t, userBiz.lastUpdateRequest.Nickname)
			assert.Equal(t, tt.wantNickname, userBiz.lastUpdateRequest.GetNickname())
		})
	}
}

func TestChangePasswordBindsPathUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userBiz := &stubUserBiz{}
	h := NewHandler(stubBiz{userBiz: userBiz}, &validation.Validator{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.Background()
		ctx = contextx.WithUsername(ctx, "ordinary-user")
		ctx = contextx.WithUserID(ctx, "path-user-id")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.PUT("/v1/users/:userID/change-password", h.ChangePassword)

	req := httptest.NewRequest(http.MethodPut, "/v1/users/path-user-id/change-password", bytes.NewBufferString(`{"oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, userBiz.lastChangePasswordRequest)
	assert.Equal(t, "path-user-id", userBiz.lastChangePasswordRequest.GetUserID())
	assert.Equal(t, "abc123", userBiz.lastChangePasswordRequest.GetOldPassword())
	assert.Equal(t, "xyz789", userBiz.lastChangePasswordRequest.GetNewPassword())
}

func TestAssignRolesToUserBindsPathUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRoleBiz := &stubUserRoleBiz{}
	h := NewHandler(stubBiz{userRoleBiz: userRoleBiz}, &validation.Validator{})

	r := gin.New()
	r.POST("/v1/users/:userID/roles", h.AssignRolesToUser)

	req := httptest.NewRequest(http.MethodPost, "/v1/users/path-user-id/roles", bytes.NewBufferString(`{"userID":"body-user-id","roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, userRoleBiz.lastAssignRolesRequest)
	assert.Equal(t, "path-user-id", userRoleBiz.lastAssignRolesRequest.GetUserID())
	assert.Equal(t, []string{"role-1", "role-2"}, userRoleBiz.lastAssignRolesRequest.GetRoleIDs())
}

func TestToggleScheduledTaskBindsPathScheduledTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	scheduledTaskBiz := &stubScheduledTaskBiz{}
	h := NewHandler(stubBiz{scheduledTaskBiz: scheduledTaskBiz}, &validation.Validator{})

	r := gin.New()
	r.PUT("/v1/scheduled-tasks/:scheduledTaskID/toggle", h.ToggleScheduledTask)

	req := httptest.NewRequest(http.MethodPut, "/v1/scheduled-tasks/task-from-path/toggle", bytes.NewBufferString(`{"scheduledTaskID":"task-from-body","enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, scheduledTaskBiz.lastToggleRequest)
	assert.Equal(t, "task-from-path", scheduledTaskBiz.lastToggleRequest.GetScheduledTaskID())
	assert.True(t, scheduledTaskBiz.lastToggleRequest.GetEnabled())
}

func TestAssignPermissionsToRoleBindsPathRoleID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	roleBiz := &stubRoleBiz{}
	h := NewHandler(stubBiz{roleBiz: roleBiz}, &validation.Validator{})

	r := gin.New()
	r.POST("/v1/roles/:roleID/permissions", h.AssignPermissionsToRole)

	req := httptest.NewRequest(http.MethodPost, "/v1/roles/path-role-id/permissions", bytes.NewBufferString(`{"roleID":"body-role-id","permissionIDs":["perm-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, roleBiz.lastAssignPermissionsRequest)
	assert.Equal(t, "path-role-id", roleBiz.lastAssignPermissionsRequest.GetRoleID())
	assert.Equal(t, []string{"perm-1"}, roleBiz.lastAssignPermissionsRequest.GetPermissionIDs())
}

func TestUpdateMenuBindsPathMenuID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	menuBiz := &stubMenuBiz{}
	h := NewHandler(stubBiz{menuBiz: menuBiz}, &validation.Validator{})

	r := gin.New()
	r.PUT("/v1/menus/:menuID", h.UpdateMenu)

	req := httptest.NewRequest(http.MethodPut, "/v1/menus/path-menu-id", bytes.NewBufferString(`{"menuID":"body-menu-id","menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, menuBiz.lastUpdateRequest)
	assert.Equal(t, "path-menu-id", menuBiz.lastUpdateRequest.GetMenuID())
	assert.Equal(t, "仪表盘", menuBiz.lastUpdateRequest.GetMenuName())
}

func TestUpdateRoleBindsPathRoleID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	roleBiz := &stubRoleBiz{}
	h := NewHandler(stubBiz{roleBiz: roleBiz}, &validation.Validator{})

	r := gin.New()
	r.PUT("/v1/roles/:roleID", h.UpdateRole)

	req := httptest.NewRequest(http.MethodPut, "/v1/roles/path-role-id", bytes.NewBufferString(`{"roleID":"body-role-id","roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, roleBiz.lastUpdateRequest)
	assert.Equal(t, "path-role-id", roleBiz.lastUpdateRequest.GetRoleID())
	assert.Equal(t, "管理员", roleBiz.lastUpdateRequest.GetRoleName())
}

func TestUpdatePermissionBindsPathPermissionID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	permissionBiz := &stubPermissionBiz{}
	h := NewHandler(stubBiz{permissionBiz: permissionBiz}, &validation.Validator{})

	r := gin.New()
	r.PUT("/v1/permissions/:permissionID", h.UpdatePermission)

	req := httptest.NewRequest(http.MethodPut, "/v1/permissions/path-permission-id", bytes.NewBufferString(`{"permissionID":"body-permission-id","permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, permissionBiz.lastUpdateRequest)
	assert.Equal(t, "path-permission-id", permissionBiz.lastUpdateRequest.GetPermissionID())
	assert.Equal(t, "用户查询", permissionBiz.lastUpdateRequest.GetPermissionName())
}

func TestHandleUriJSONRequestPrefersURIOverBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID" uri:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"userID":"body-user-id","nickname":"neo"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "neo", *got.Nickname)
}

func TestHandleUriJSONRequestBindsBodyAndURI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID      string `json:"userID" uri:"userID"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID/change-password", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ChangePasswordResponse, error) {
			got = rq
			return &v1.ChangePasswordResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id/change-password", bytes.NewBufferString(`{"oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, "abc123", got.OldPassword)
	assert.Equal(t, "xyz789", got.NewPassword)
}

func TestHandleUriJSONRequestReturnsBindErrorWhenURIFieldMissingTag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"nickname":"neo"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "", got.UserID)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "neo", *got.Nickname)
}

func TestHandleUriJSONRequestAllowsBodyOnlyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string `json:"userID" uri:"userID"`
		Enabled bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestPreservesBodyOptionalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID" uri:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"nickname":"trinity"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "trinity", *got.Nickname)
}

func TestHandleUriJSONRequestSupportsEmptyBodyWithURI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID string `json:"userID" uri:"userID"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
}

func TestHandleUriJSONRequestIgnoresBodyUserIDWhenURIExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID string `json:"userID" uri:"userID"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"userID":"body-user-id"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
}

func TestHandleUriJSONRequestPreservesBooleanBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		ScheduledTaskID string `json:"scheduledTaskID" uri:"scheduledTaskID"`
		Enabled         bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:scheduledTaskID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/task-id", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "task-id", got.ScheduledTaskID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestPreservesRepeatedBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string   `json:"userID" uri:"userID"`
		RoleIDs []string `json:"roleIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:userID/roles", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignRolesToUserResponse, error) {
			got = rq
			return &v1.AssignRolesToUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-user-id/roles", bytes.NewBufferString(`{"roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, []string{"role-1", "role-2"}, got.RoleIDs)
}

func TestHandleUriJSONRequestPreservesRepeatedBodyFieldWhenBodyContainsConflictingID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID        string   `json:"roleID" uri:"roleID"`
		PermissionIDs []string `json:"permissionIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:roleID/permissions", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignPermissionsToRoleResponse, error) {
			got = rq
			return &v1.AssignPermissionsToRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-role-id/permissions", bytes.NewBufferString(`{"roleID":"body-role-id","permissionIDs":["perm-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, []string{"perm-1"}, got.PermissionIDs)
}

func TestHandleUriJSONRequestPreservesStringBodyFieldWithURIOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		PermissionID   string `json:"permissionID" uri:"permissionID"`
		PermissionName string `json:"permissionName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:permissionID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdatePermissionResponse, error) {
			got = rq
			return &v1.UpdatePermissionResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-permission-id", bytes.NewBufferString(`{"permissionID":"body-permission-id","permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-permission-id", got.PermissionID)
	assert.Equal(t, "用户查询", got.PermissionName)
}

func TestHandleUriJSONRequestPreservesStringBodyFieldForMenuUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		MenuID   string `json:"menuID" uri:"menuID"`
		MenuName string `json:"menuName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:menuID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateMenuResponse, error) {
			got = rq
			return &v1.UpdateMenuResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-menu-id", bytes.NewBufferString(`{"menuID":"body-menu-id","menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-menu-id", got.MenuID)
	assert.Equal(t, "仪表盘", got.MenuName)
}

func TestHandleUriJSONRequestPreservesStringBodyFieldForRoleUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID   string `json:"roleID" uri:"roleID"`
		RoleName string `json:"roleName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:roleID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateRoleResponse, error) {
			got = rq
			return &v1.UpdateRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-role-id", bytes.NewBufferString(`{"roleID":"body-role-id","roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, "管理员", got.RoleName)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWhenBodyHasSameField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID string `json:"userID" uri:"userID"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"userID":"body-user-id"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
}

func TestHandleUriJSONRequestUsesPathIDForChangePasswordWhenBodyOmitsUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID      string `json:"userID" uri:"userID"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID/change-password", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ChangePasswordResponse, error) {
			got = rq
			return &v1.ChangePasswordResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id/change-password", bytes.NewBufferString(`{"oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, "abc123", got.OldPassword)
	assert.Equal(t, "xyz789", got.NewPassword)
}

func TestHandleUriJSONRequestUsesPathIDForScheduledTaskToggleWhenBodyConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		ScheduledTaskID string `json:"scheduledTaskID" uri:"scheduledTaskID"`
		Enabled         bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:scheduledTaskID/toggle", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/task-from-path/toggle", bytes.NewBufferString(`{"scheduledTaskID":"task-from-body","enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "task-from-path", got.ScheduledTaskID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestUsesPathIDForAssignRolesWhenBodyConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string   `json:"userID" uri:"userID"`
		RoleIDs []string `json:"roleIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:userID/roles", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignRolesToUserResponse, error) {
			got = rq
			return &v1.AssignRolesToUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-user-id/roles", bytes.NewBufferString(`{"userID":"body-user-id","roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, []string{"role-1", "role-2"}, got.RoleIDs)
}

func TestHandleUriJSONRequestUsesPathIDForAssignPermissionsWhenBodyConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID        string   `json:"roleID" uri:"roleID"`
		PermissionIDs []string `json:"permissionIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:roleID/permissions", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignPermissionsToRoleResponse, error) {
			got = rq
			return &v1.AssignPermissionsToRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-role-id/permissions", bytes.NewBufferString(`{"roleID":"body-role-id","permissionIDs":["perm-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, []string{"perm-1"}, got.PermissionIDs)
}

func TestHandleUriJSONRequestUsesPathIDForPermissionUpdateWhenBodyConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		PermissionID   string `json:"permissionID" uri:"permissionID"`
		PermissionName string `json:"permissionName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:permissionID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdatePermissionResponse, error) {
			got = rq
			return &v1.UpdatePermissionResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-permission-id", bytes.NewBufferString(`{"permissionID":"body-permission-id","permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-permission-id", got.PermissionID)
	assert.Equal(t, "用户查询", got.PermissionName)
}

func TestHandleUriJSONRequestUsesPathIDForMenuUpdateWhenBodyConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		MenuID   string `json:"menuID" uri:"menuID"`
		MenuName string `json:"menuName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:menuID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateMenuResponse, error) {
			got = rq
			return &v1.UpdateMenuResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-menu-id", bytes.NewBufferString(`{"menuID":"body-menu-id","menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-menu-id", got.MenuID)
	assert.Equal(t, "仪表盘", got.MenuName)
}

func TestHandleUriJSONRequestUsesPathIDForRoleUpdateWhenBodyConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID   string `json:"roleID" uri:"roleID"`
		RoleName string `json:"roleName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:roleID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateRoleResponse, error) {
			got = rq
			return &v1.UpdateRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-role-id", bytes.NewBufferString(`{"roleID":"body-role-id","roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, "管理员", got.RoleName)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWhenBodyOmitsOtherFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID string `json:"userID" uri:"userID"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
}

func TestHandleUriJSONRequestUsesPathIDForAssignPermissionsWithBodyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID        string   `json:"roleID" uri:"roleID"`
		PermissionIDs []string `json:"permissionIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:roleID/permissions", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignPermissionsToRoleResponse, error) {
			got = rq
			return &v1.AssignPermissionsToRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-role-id/permissions", bytes.NewBufferString(`{"permissionIDs":["perm-1","perm-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, []string{"perm-1", "perm-2"}, got.PermissionIDs)
}

func TestHandleUriJSONRequestUsesPathIDForAssignRolesWithBodyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string   `json:"userID" uri:"userID"`
		RoleIDs []string `json:"roleIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:userID/roles", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignRolesToUserResponse, error) {
			got = rq
			return &v1.AssignRolesToUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-user-id/roles", bytes.NewBufferString(`{"roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, []string{"role-1", "role-2"}, got.RoleIDs)
}

func TestHandleUriJSONRequestUsesPathIDForScheduledTaskToggleWithBodyOnlyEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		ScheduledTaskID string `json:"scheduledTaskID" uri:"scheduledTaskID"`
		Enabled         bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:scheduledTaskID/toggle", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/task-id/toggle", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "task-id", got.ScheduledTaskID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestUsesPathIDForChangePasswordWithBodyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID      string `json:"userID" uri:"userID"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID/change-password", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ChangePasswordResponse, error) {
			got = rq
			return &v1.ChangePasswordResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id/change-password", bytes.NewBufferString(`{"oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, "abc123", got.OldPassword)
	assert.Equal(t, "xyz789", got.NewPassword)
}

func TestHandleUriJSONRequestUsesPathIDForPermissionUpdateWithBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		PermissionID   string `json:"permissionID" uri:"permissionID"`
		PermissionName string `json:"permissionName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:permissionID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdatePermissionResponse, error) {
			got = rq
			return &v1.UpdatePermissionResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-permission-id", bytes.NewBufferString(`{"permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-permission-id", got.PermissionID)
	assert.Equal(t, "用户查询", got.PermissionName)
}

func TestHandleUriJSONRequestUsesPathIDForMenuUpdateWithBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		MenuID   string `json:"menuID" uri:"menuID"`
		MenuName string `json:"menuName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:menuID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateMenuResponse, error) {
			got = rq
			return &v1.UpdateMenuResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-menu-id", bytes.NewBufferString(`{"menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-menu-id", got.MenuID)
	assert.Equal(t, "仪表盘", got.MenuName)
}

func TestHandleUriJSONRequestUsesPathIDForRoleUpdateWithBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID   string `json:"roleID" uri:"roleID"`
		RoleName string `json:"roleName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:roleID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateRoleResponse, error) {
			got = rq
			return &v1.UpdateRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-role-id", bytes.NewBufferString(`{"roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, "管理员", got.RoleName)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWithBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID" uri:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"nickname":"neo"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "neo", *got.Nickname)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWithConflictingBodyAndOptionalField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID" uri:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"userID":"body-user-id","nickname":"trinity"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "trinity", *got.Nickname)
}

func TestHandleUriJSONRequestUsesPathIDForAssignRolesWithConflictingBodyAndRepeatedField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string   `json:"userID" uri:"userID"`
		RoleIDs []string `json:"roleIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:userID/roles", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignRolesToUserResponse, error) {
			got = rq
			return &v1.AssignRolesToUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-user-id/roles", bytes.NewBufferString(`{"userID":"body-user-id","roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, []string{"role-1", "role-2"}, got.RoleIDs)
}

func TestHandleUriJSONRequestUsesPathIDForAssignPermissionsWithConflictingBodyAndRepeatedField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID        string   `json:"roleID" uri:"roleID"`
		PermissionIDs []string `json:"permissionIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:roleID/permissions", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignPermissionsToRoleResponse, error) {
			got = rq
			return &v1.AssignPermissionsToRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-role-id/permissions", bytes.NewBufferString(`{"roleID":"body-role-id","permissionIDs":["perm-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, []string{"perm-1"}, got.PermissionIDs)
}

func TestHandleUriJSONRequestUsesPathIDForScheduledTaskToggleWithConflictingBodyAndBoolField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		ScheduledTaskID string `json:"scheduledTaskID" uri:"scheduledTaskID"`
		Enabled         bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:scheduledTaskID/toggle", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/task-from-path/toggle", bytes.NewBufferString(`{"scheduledTaskID":"task-from-body","enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "task-from-path", got.ScheduledTaskID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestUsesPathIDForChangePasswordWithConflictingBodyIDAndStringFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID      string `json:"userID" uri:"userID"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID/change-password", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ChangePasswordResponse, error) {
			got = rq
			return &v1.ChangePasswordResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id/change-password", bytes.NewBufferString(`{"userID":"body-user-id","oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, "abc123", got.OldPassword)
	assert.Equal(t, "xyz789", got.NewPassword)
}

func TestHandleUriJSONRequestUsesPathIDForPermissionUpdateWithConflictingBodyIDAndStringField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		PermissionID   string `json:"permissionID" uri:"permissionID"`
		PermissionName string `json:"permissionName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:permissionID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdatePermissionResponse, error) {
			got = rq
			return &v1.UpdatePermissionResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-permission-id", bytes.NewBufferString(`{"permissionID":"body-permission-id","permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-permission-id", got.PermissionID)
	assert.Equal(t, "用户查询", got.PermissionName)
}

func TestHandleUriJSONRequestUsesPathIDForMenuUpdateWithConflictingBodyIDAndStringField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		MenuID   string `json:"menuID" uri:"menuID"`
		MenuName string `json:"menuName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:menuID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateMenuResponse, error) {
			got = rq
			return &v1.UpdateMenuResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-menu-id", bytes.NewBufferString(`{"menuID":"body-menu-id","menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-menu-id", got.MenuID)
	assert.Equal(t, "仪表盘", got.MenuName)
}

func TestHandleUriJSONRequestUsesPathIDForRoleUpdateWithConflictingBodyIDAndStringField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID   string `json:"roleID" uri:"roleID"`
		RoleName string `json:"roleName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:roleID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateRoleResponse, error) {
			got = rq
			return &v1.UpdateRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-role-id", bytes.NewBufferString(`{"roleID":"body-role-id","roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, "管理员", got.RoleName)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWithConflictingBodyIDAndOptionalField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID" uri:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"userID":"body-user-id","nickname":"trinity"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "trinity", *got.Nickname)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWithOptionalBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID   string  `json:"userID" uri:"userID"`
		Nickname *string `json:"nickname,omitempty"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{"nickname":"neo"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	require.NotNil(t, got.Nickname)
	assert.Equal(t, "neo", *got.Nickname)
}

func TestHandleUriJSONRequestUsesPathIDForChangePasswordWithStringBodyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID      string `json:"userID" uri:"userID"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID/change-password", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ChangePasswordResponse, error) {
			got = rq
			return &v1.ChangePasswordResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id/change-password", bytes.NewBufferString(`{"oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, "abc123", got.OldPassword)
	assert.Equal(t, "xyz789", got.NewPassword)
}

func TestHandleUriJSONRequestUsesPathIDForScheduledTaskToggleWithBoolBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		ScheduledTaskID string `json:"scheduledTaskID" uri:"scheduledTaskID"`
		Enabled         bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:scheduledTaskID/toggle", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/task-id/toggle", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "task-id", got.ScheduledTaskID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestUsesPathIDForAssignRolesWithRepeatedBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string   `json:"userID" uri:"userID"`
		RoleIDs []string `json:"roleIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:userID/roles", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignRolesToUserResponse, error) {
			got = rq
			return &v1.AssignRolesToUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-user-id/roles", bytes.NewBufferString(`{"roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, []string{"role-1", "role-2"}, got.RoleIDs)
}

func TestHandleUriJSONRequestUsesPathIDForAssignPermissionsWithRepeatedBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID        string   `json:"roleID" uri:"roleID"`
		PermissionIDs []string `json:"permissionIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:roleID/permissions", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignPermissionsToRoleResponse, error) {
			got = rq
			return &v1.AssignPermissionsToRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-role-id/permissions", bytes.NewBufferString(`{"permissionIDs":["perm-1","perm-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, []string{"perm-1", "perm-2"}, got.PermissionIDs)
}

func TestHandleUriJSONRequestUsesPathIDForMenuUpdateWithStringBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		MenuID   string `json:"menuID" uri:"menuID"`
		MenuName string `json:"menuName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:menuID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateMenuResponse, error) {
			got = rq
			return &v1.UpdateMenuResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-menu-id", bytes.NewBufferString(`{"menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-menu-id", got.MenuID)
	assert.Equal(t, "仪表盘", got.MenuName)
}

func TestHandleUriJSONRequestUsesPathIDForRoleUpdateWithStringBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID   string `json:"roleID" uri:"roleID"`
		RoleName string `json:"roleName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:roleID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateRoleResponse, error) {
			got = rq
			return &v1.UpdateRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-role-id", bytes.NewBufferString(`{"roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, "管理员", got.RoleName)
}

func TestHandleUriJSONRequestUsesPathIDForPermissionUpdateWithStringBodyField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		PermissionID   string `json:"permissionID" uri:"permissionID"`
		PermissionName string `json:"permissionName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:permissionID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdatePermissionResponse, error) {
			got = rq
			return &v1.UpdatePermissionResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-permission-id", bytes.NewBufferString(`{"permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-permission-id", got.PermissionID)
	assert.Equal(t, "用户查询", got.PermissionName)
}

func TestHandleUriJSONRequestUsesPathIDForUserUpdateWithEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID string `json:"userID" uri:"userID"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateUserResponse, error) {
			got = rq
			return &v1.UpdateUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
}

func TestHandleUriJSONRequestUsesPathIDForChangePasswordWithBodyOnlyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID      string `json:"userID" uri:"userID"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:userID/change-password", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ChangePasswordResponse, error) {
			got = rq
			return &v1.ChangePasswordResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-user-id/change-password", bytes.NewBufferString(`{"oldPassword":"abc123","newPassword":"xyz789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, "abc123", got.OldPassword)
	assert.Equal(t, "xyz789", got.NewPassword)
}

func TestHandleUriJSONRequestUsesPathIDForScheduledTaskToggleWithEmptyBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		ScheduledTaskID string `json:"scheduledTaskID" uri:"scheduledTaskID"`
		Enabled         bool   `json:"enabled"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:scheduledTaskID/toggle", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.ToggleScheduledTaskResponse, error) {
			got = rq
			return &v1.ToggleScheduledTaskResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/task-id/toggle", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "task-id", got.ScheduledTaskID)
	assert.True(t, got.Enabled)
}

func TestHandleUriJSONRequestUsesPathIDForAssignRolesWithEmptyBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		UserID  string   `json:"userID" uri:"userID"`
		RoleIDs []string `json:"roleIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:userID/roles", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignRolesToUserResponse, error) {
			got = rq
			return &v1.AssignRolesToUserResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-user-id/roles", bytes.NewBufferString(`{"roleIDs":["role-1","role-2"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-user-id", got.UserID)
	assert.Equal(t, []string{"role-1", "role-2"}, got.RoleIDs)
}

func TestHandleUriJSONRequestUsesPathIDForAssignPermissionsWithEmptyBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID        string   `json:"roleID" uri:"roleID"`
		PermissionIDs []string `json:"permissionIDs"`
	}

	var got *request
	r := gin.New()
	r.POST("/tests/:roleID/permissions", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.AssignPermissionsToRoleResponse, error) {
			got = rq
			return &v1.AssignPermissionsToRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/tests/path-role-id/permissions", bytes.NewBufferString(`{"permissionIDs":["perm-1"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, []string{"perm-1"}, got.PermissionIDs)
}

func TestHandleUriJSONRequestUsesPathIDForPermissionUpdateWithEmptyBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		PermissionID   string `json:"permissionID" uri:"permissionID"`
		PermissionName string `json:"permissionName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:permissionID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdatePermissionResponse, error) {
			got = rq
			return &v1.UpdatePermissionResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-permission-id", bytes.NewBufferString(`{"permissionName":"用户查询"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-permission-id", got.PermissionID)
	assert.Equal(t, "用户查询", got.PermissionName)
}

func TestHandleUriJSONRequestUsesPathIDForMenuUpdateWithEmptyBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		MenuID   string `json:"menuID" uri:"menuID"`
		MenuName string `json:"menuName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:menuID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateMenuResponse, error) {
			got = rq
			return &v1.UpdateMenuResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-menu-id", bytes.NewBufferString(`{"menuName":"仪表盘"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-menu-id", got.MenuID)
	assert.Equal(t, "仪表盘", got.MenuName)
}

func TestHandleUriJSONRequestUsesPathIDForRoleUpdateWithEmptyBodyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type request struct {
		RoleID   string `json:"roleID" uri:"roleID"`
		RoleName string `json:"roleName"`
	}

	var got *request
	r := gin.New()
	r.PUT("/tests/:roleID", func(c *gin.Context) {
		core.HandleUriJSONRequest(c, func(ctx context.Context, rq *request) (*v1.UpdateRoleResponse, error) {
			got = rq
			return &v1.UpdateRoleResponse{}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPut, "/tests/path-role-id", bytes.NewBufferString(`{"roleName":"管理员"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, got)
	assert.Equal(t, "path-role-id", got.RoleID)
	assert.Equal(t, "管理员", got.RoleName)
}

type stubBiz struct {
	userBiz          userv1.UserBiz
	roleBiz          rolev1.RoleBiz
	permissionBiz    permissionv1.PermissionBiz
	menuBiz          menuv1.MenuBiz
	userRoleBiz      userrolev1.UserRoleBiz
	scheduledTaskBiz scheduledtaskv1.ScheduledTaskBiz
}

func (b stubBiz) UserV1() userv1.UserBiz { return b.userBiz }

func (b stubBiz) RoleV1() rolev1.RoleBiz { return b.roleBiz }

func (b stubBiz) PermissionV1() permissionv1.PermissionBiz { return b.permissionBiz }

func (b stubBiz) MenuV1() menuv1.MenuBiz { return b.menuBiz }

func (b stubBiz) UserRoleV1() userrolev1.UserRoleBiz { return b.userRoleBiz }

func (b stubBiz) ScheduledTaskV1() scheduledtaskv1.ScheduledTaskBiz { return b.scheduledTaskBiz }

type stubUserBiz struct {
	lastUpdateRequest         *v1.UpdateUserRequest
	lastChangePasswordRequest *v1.ChangePasswordRequest
}

func (b *stubUserBiz) Create(context.Context, *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	panic("unexpected call to Create")
}

func (b *stubUserBiz) Update(_ context.Context, rq *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	b.lastUpdateRequest = rq
	return &v1.UpdateUserResponse{}, nil
}

func (b *stubUserBiz) Delete(context.Context, *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	panic("unexpected call to Delete")
}

func (b *stubUserBiz) Get(context.Context, *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	panic("unexpected call to Get")
}

func (b *stubUserBiz) List(context.Context, *v1.ListUserRequest) (*v1.ListUserResponse, error) {
	panic("unexpected call to List")
}

func (b *stubUserBiz) Login(context.Context, *v1.LoginRequest) (*v1.LoginResponse, error) {
	panic("unexpected call to Login")
}

func (b *stubUserBiz) RefreshToken(context.Context, *v1.RefreshTokenRequest) (*v1.LoginResponse, error) {
	panic("unexpected call to RefreshToken")
}

func (b *stubUserBiz) ChangePassword(_ context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error) {
	b.lastChangePasswordRequest = rq
	return &v1.ChangePasswordResponse{}, nil
}

type stubUserRoleBiz struct {
	lastAssignRolesRequest *v1.AssignRolesToUserRequest
}

func (b *stubUserRoleBiz) AssignRolesToUser(_ context.Context, rq *v1.AssignRolesToUserRequest) (*v1.AssignRolesToUserResponse, error) {
	b.lastAssignRolesRequest = rq
	return &v1.AssignRolesToUserResponse{}, nil
}

func (b *stubUserRoleBiz) GetUserRoles(context.Context, *v1.GetUserRolesRequest) (*v1.GetUserRolesResponse, error) {
	panic("unexpected call to GetUserRoles")
}

func (b *stubUserRoleBiz) RemoveRoleFromUser(context.Context, *v1.RemoveRoleFromUserRequest) (*v1.RemoveRoleFromUserResponse, error) {
	panic("unexpected call to RemoveRoleFromUser")
}

type stubScheduledTaskBiz struct {
	lastToggleRequest *v1.ToggleScheduledTaskRequest
}

func (b *stubScheduledTaskBiz) Create(context.Context, *v1.CreateScheduledTaskRequest) (*v1.CreateScheduledTaskResponse, error) {
	panic("unexpected call to Create")
}

func (b *stubScheduledTaskBiz) Update(context.Context, *v1.UpdateScheduledTaskRequest) (*v1.UpdateScheduledTaskResponse, error) {
	panic("unexpected call to Update")
}

func (b *stubScheduledTaskBiz) Delete(context.Context, *v1.DeleteScheduledTaskRequest) (*v1.DeleteScheduledTaskResponse, error) {
	panic("unexpected call to Delete")
}

func (b *stubScheduledTaskBiz) Get(context.Context, *v1.GetScheduledTaskRequest) (*v1.GetScheduledTaskResponse, error) {
	panic("unexpected call to Get")
}

func (b *stubScheduledTaskBiz) List(context.Context, *v1.ListScheduledTasksRequest) (*v1.ListScheduledTasksResponse, error) {
	panic("unexpected call to List")
}

func (b *stubScheduledTaskBiz) Toggle(_ context.Context, rq *v1.ToggleScheduledTaskRequest) (*v1.ToggleScheduledTaskResponse, error) {
	b.lastToggleRequest = rq
	return &v1.ToggleScheduledTaskResponse{}, nil
}

func (b *stubScheduledTaskBiz) Trigger(context.Context, *v1.TriggerScheduledTaskRequest) (*v1.TriggerScheduledTaskResponse, error) {
	panic("unexpected call to Trigger")
}

func (b *stubScheduledTaskBiz) ListExecutions(context.Context, *v1.ListScheduledTaskExecutionsRequest) (*v1.ListScheduledTaskExecutionsResponse, error) {
	panic("unexpected call to ListExecutions")
}

type stubRoleBiz struct {
	lastUpdateRequest            *v1.UpdateRoleRequest
	lastAssignPermissionsRequest *v1.AssignPermissionsToRoleRequest
}

func (b *stubRoleBiz) Create(context.Context, *v1.CreateRoleRequest) (*v1.CreateRoleResponse, error) {
	panic("unexpected call to Create")
}

func (b *stubRoleBiz) Update(_ context.Context, rq *v1.UpdateRoleRequest) (*v1.UpdateRoleResponse, error) {
	b.lastUpdateRequest = rq
	return &v1.UpdateRoleResponse{}, nil
}

func (b *stubRoleBiz) Delete(context.Context, *v1.DeleteRoleRequest) (*v1.DeleteRoleResponse, error) {
	panic("unexpected call to Delete")
}

func (b *stubRoleBiz) Get(context.Context, *v1.GetRoleRequest) (*v1.GetRoleResponse, error) {
	panic("unexpected call to Get")
}

func (b *stubRoleBiz) List(context.Context, *v1.ListRoleRequest) (*v1.ListRoleResponse, error) {
	panic("unexpected call to List")
}

func (b *stubRoleBiz) AssignPermissionsToRole(_ context.Context, rq *v1.AssignPermissionsToRoleRequest) (*v1.AssignPermissionsToRoleResponse, error) {
	b.lastAssignPermissionsRequest = rq
	return &v1.AssignPermissionsToRoleResponse{}, nil
}

func (b *stubRoleBiz) GetRolePermissions(context.Context, *v1.GetRolePermissionsRequest) (*v1.GetRolePermissionsResponse, error) {
	panic("unexpected call to GetRolePermissions")
}

type stubMenuBiz struct {
	lastUpdateRequest *v1.UpdateMenuRequest
}

func (b *stubMenuBiz) Create(context.Context, *v1.CreateMenuRequest) (*v1.CreateMenuResponse, error) {
	panic("unexpected call to Create")
}

func (b *stubMenuBiz) Update(_ context.Context, rq *v1.UpdateMenuRequest) (*v1.UpdateMenuResponse, error) {
	b.lastUpdateRequest = rq
	return &v1.UpdateMenuResponse{}, nil
}

func (b *stubMenuBiz) Delete(context.Context, *v1.DeleteMenuRequest) (*v1.DeleteMenuResponse, error) {
	panic("unexpected call to Delete")
}

func (b *stubMenuBiz) Get(context.Context, *v1.GetMenuRequest) (*v1.GetMenuResponse, error) {
	panic("unexpected call to Get")
}

func (b *stubMenuBiz) List(context.Context, *v1.ListMenuRequest) (*v1.ListMenuResponse, error) {
	panic("unexpected call to List")
}

func (b *stubMenuBiz) ListMenuTree(context.Context, *v1.ListMenuTreeRequest) (*v1.ListMenuTreeResponse, error) {
	panic("unexpected call to ListMenuTree")
}

func (b *stubMenuBiz) GetUserMenuTree(context.Context, *v1.GetUserMenuTreeRequest) (*v1.GetUserMenuTreeResponse, error) {
	panic("unexpected call to GetUserMenuTree")
}

func (b *stubMenuBiz) GetMenuRoles(context.Context, *v1.GetMenuRolesRequest) (*v1.GetMenuRolesResponse, error) {
	panic("unexpected call to GetMenuRoles")
}

func (b *stubMenuBiz) SetMenuRoles(context.Context, *v1.SetMenuRolesRequest) (*v1.SetMenuRolesResponse, error) {
	panic("unexpected call to SetMenuRoles")
}

func (b *stubMenuBiz) AddMenuRole(context.Context, *v1.AddMenuRoleRequest) (*v1.AddMenuRoleResponse, error) {
	panic("unexpected call to AddMenuRole")
}

func (b *stubMenuBiz) RemoveMenuRole(context.Context, *v1.RemoveMenuRoleRequest) (*v1.RemoveMenuRoleResponse, error) {
	panic("unexpected call to RemoveMenuRole")
}

type stubPermissionBiz struct {
	lastUpdateRequest *v1.UpdatePermissionRequest
}

func (b *stubPermissionBiz) Create(context.Context, *v1.CreatePermissionRequest) (*v1.CreatePermissionResponse, error) {
	panic("unexpected call to Create")
}

func (b *stubPermissionBiz) Update(_ context.Context, rq *v1.UpdatePermissionRequest) (*v1.UpdatePermissionResponse, error) {
	b.lastUpdateRequest = rq
	return &v1.UpdatePermissionResponse{}, nil
}

func (b *stubPermissionBiz) Delete(context.Context, *v1.DeletePermissionRequest) (*v1.DeletePermissionResponse, error) {
	panic("unexpected call to Delete")
}

func (b *stubPermissionBiz) Get(context.Context, *v1.GetPermissionRequest) (*v1.GetPermissionResponse, error) {
	panic("unexpected call to Get")
}

func (b *stubPermissionBiz) List(context.Context, *v1.ListPermissionRequest) (*v1.ListPermissionResponse, error) {
	panic("unexpected call to List")
}

func (b *stubPermissionBiz) ListPermissionTree(context.Context, *v1.ListPermissionTreeRequest) (*v1.ListPermissionTreeResponse, error) {
	panic("unexpected call to ListPermissionTree")
}
