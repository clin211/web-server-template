package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

var (
	// ErrRoleAlreadyExists 角色已存在
	ErrRoleAlreadyExists = errorsx.NewCompat(409, "Role.AlreadyExists", "Role already exists.")

	// ErrRoleNotFound 角色不存在
	ErrRoleNotFound = errorsx.NewCompat(404, "Role.NotFound", "Role not found.")

	// ErrPermissionAlreadyExists 权限已存在
	ErrPermissionAlreadyExists = errorsx.NewCompat(409, "Permission.AlreadyExists", "Permission already exists.")

	// ErrPermissionNotFound 权限不存在
	ErrPermissionNotFound = errorsx.NewCompat(404, "Permission.NotFound", "Permission not found.")

	// ErrPermissionHasChildren 权限有子权限
	ErrPermissionHasChildren = errorsx.NewCompat(400, "Permission.HasChildren", "Permission has children, cannot delete.")

	// ErrMenuAlreadyExists 菜单已存在
	ErrMenuAlreadyExists = errorsx.NewCompat(409, "Menu.AlreadyExists", "Menu already exists.")

	// ErrMenuNotFound 菜单不存在
	ErrMenuNotFound = errorsx.NewCompat(404, "Menu.NotFound", "Menu not found.")

	// ErrMenuHasChildren 菜单有子菜单
	ErrMenuHasChildren = errorsx.NewCompat(400, "Menu.HasChildren", "Menu has children, cannot delete.")
)
