package conversion

import (
	"fmt"
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

var menuConversionResult *v1.Menu
var menuListConversionResult []*v1.Menu
var menuTreeConversionResult []*v1.MenuTreeNode

func BenchmarkMenuModelToMenuV1(b *testing.B) {
	parentID := "menu-parent"
	icon := "user"
	path := "/system/user"
	component := "system/user/index"
	permissionID := "perm-1"
	menu := &model.MenuM{
		MenuID:       "menu-1",
		ParentID:     &parentID,
		MenuName:     "用户管理",
		MenuCode:     "user",
		MenuType:     "page",
		Icon:         &icon,
		Path:         &path,
		Component:    &component,
		PermissionID: &permissionID,
		SortOrder:    10,
		Visible:      1,
		Status:       0,
		CreatedAt:    time.Unix(1700000000, 0),
		UpdatedAt:    time.Unix(1700003600, 0),
	}

	b.ReportAllocs()
	for b.Loop() {
		menuConversionResult = MenuModelToMenuV1(menu)
	}
}

func BenchmarkMenuModelListToMenuV1List(b *testing.B) {
	menus := buildBenchmarkMenus(100)

	b.ReportAllocs()
	for b.Loop() {
		menuListConversionResult = MenuModelListToMenuV1List(menus)
	}
}

func BenchmarkMenuModelListToMenuTreeV1(b *testing.B) {
	menus := buildBenchmarkMenus(100)

	b.ReportAllocs()
	for b.Loop() {
		menuTreeConversionResult = MenuModelListToMenuTreeV1(menus)
	}
}

func buildBenchmarkMenus(size int) []*model.MenuM {
	menus := make([]*model.MenuM, 0, size)
	for i := 0; i < size; i++ {
		menuID := fmt.Sprintf("menu-%d", i)
		menuType := "page"
		if i%10 == 0 {
			menuType = "menu"
		}
		menu := &model.MenuM{
			MenuID:    menuID,
			MenuName:  fmt.Sprintf("菜单-%d", i),
			MenuCode:  fmt.Sprintf("menu:%d", i),
			MenuType:  menuType,
			SortOrder: int32(i),
			Visible:   1,
			Status:    0,
			CreatedAt: time.Unix(1700000000+int64(i), 0),
			UpdatedAt: time.Unix(1700003600+int64(i), 0),
		}
		if i%10 != 0 {
			parentID := fmt.Sprintf("menu-%d", i/10*10)
			menu.ParentID = &parentID
		}
		menus = append(menus, menu)
	}
	return menus
}
