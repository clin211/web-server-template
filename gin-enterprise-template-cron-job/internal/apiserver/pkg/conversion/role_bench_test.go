package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

var roleConversionResult *v1.Role
var roleListConversionResult []*v1.Role

func BenchmarkRoleModelToRoleV1(b *testing.B) {
	description := "管理员"
	role := &model.RoleM{
		RoleID:      "role-1",
		RoleName:    "admin",
		RoleCode:    "admin",
		Description: &description,
		Status:      1,
		SortOrder:   9,
		CreatedAt:   time.Unix(1700000000, 0),
		UpdatedAt:   time.Unix(1700003600, 0),
	}

	b.ReportAllocs()
	for b.Loop() {
		roleConversionResult = RoleModelToRoleV1(role)
	}
}

func BenchmarkRoleModelListToRoleV1List(b *testing.B) {
	roles := make([]*model.RoleM, 100)
	for i := range roles {
		roles[i] = &model.RoleM{
			RoleID:    "role",
			RoleName:  "admin",
			RoleCode:  "admin",
			Status:    1,
			SortOrder: int32(i),
			CreatedAt: time.Unix(1700000000+int64(i), 0),
			UpdatedAt: time.Unix(1700003600+int64(i), 0),
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		roleListConversionResult = RoleModelListToRoleV1List(roles)
	}
}
