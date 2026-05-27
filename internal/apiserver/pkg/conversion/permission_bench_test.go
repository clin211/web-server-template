package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

var permissionConversionResult *v1.Permission
var permissionListConversionResult []*v1.Permission

func BenchmarkPermissionModelToPermissionV1(b *testing.B) {
	resourcePath := "/system/user/list"
	description := "用户列表权限"
	parentID := "perm-parent"
	path := "system.user.list"
	permission := &model.PermissionM{
		PermissionID:   "perm-1",
		PermissionName: "用户列表",
		PermissionCode: "user:list",
		ResourceType:   "button",
		ResourcePath:   &resourcePath,
		Action:         "GET",
		Description:    &description,
		ParentID:       &parentID,
		Path:           &path,
		Status:         1,
		CreatedAt:      time.Unix(1700000000, 0),
		UpdatedAt:      time.Unix(1700003600, 0),
	}

	b.ReportAllocs()
	for b.Loop() {
		permissionConversionResult = PermissionModelToPermissionV1(permission)
	}
}

func BenchmarkPermissionModelListToPermissionV1List(b *testing.B) {
	permissions := make([]*model.PermissionM, 100)
	for i := range permissions {
		permissions[i] = &model.PermissionM{
			PermissionID:   "perm",
			PermissionName: "权限",
			PermissionCode: "perm:code",
			ResourceType:   "button",
			Action:         "GET",
			Status:         1,
			CreatedAt:      time.Unix(1700000000+int64(i), 0),
			UpdatedAt:      time.Unix(1700003600+int64(i), 0),
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		permissionListConversionResult = PermissionModelListToPermissionV1List(permissions)
	}
}
