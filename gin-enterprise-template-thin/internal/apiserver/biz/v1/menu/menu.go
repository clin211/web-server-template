package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// MenuBiz 定义处理菜单请求所需的方法.
type MenuBiz interface {
	Create(ctx context.Context, rq *v1.CreateMenuRequest) (*v1.CreateMenuResponse, error)
	Update(ctx context.Context, rq *v1.UpdateMenuRequest) (*v1.UpdateMenuResponse, error)
	Delete(ctx context.Context, rq *v1.DeleteMenuRequest) (*v1.DeleteMenuResponse, error)
	Get(ctx context.Context, rq *v1.GetMenuRequest) (*v1.GetMenuResponse, error)
	List(ctx context.Context, rq *v1.ListMenuRequest) (*v1.ListMenuResponse, error)

	MenuExpansion
}

// MenuExpansion 定义菜单操作的扩展方法.
type MenuExpansion interface {
	// ListMenuTree 获取菜单树
	ListMenuTree(ctx context.Context, rq *v1.ListMenuTreeRequest) (*v1.ListMenuTreeResponse, error)
	// GetUserMenuTree 获取用户可见的菜单树
	GetUserMenuTree(ctx context.Context, rq *v1.GetUserMenuTreeRequest) (*v1.GetUserMenuTreeResponse, error)
}

// menuBiz 是 MenuBiz 接口的实现.
type menuBiz struct {
	store store.IStore
}

// 确保 menuBiz 实现了 MenuBiz 接口.
var _ MenuBiz = (*menuBiz)(nil)

func New(store store.IStore) *menuBiz {
	return &menuBiz{store: store}
}
