package biz

import (
	userv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/user"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/google/wire"
)

// ProviderSet 是 Wire 提供程序集，用于声明依赖注入规则。
// 包含用于创建 biz 实例的 NewBiz 构造函数。
// wire.Bind 将 IBiz 接口绑定到具体实现 *biz，
// 因此依赖 IBiz 的地方会自动注入 *biz 实例。
var ProviderSet = wire.NewSet(NewBiz, wire.Bind(new(IBiz), new(*biz)))

// IBiz 定义业务层必须实现的方法。
type IBiz interface {
	// UserV1 获取用户业务接口.
	UserV1() userv1.UserBiz
}

// biz 是 IBiz 的具体实现。
type biz struct {
	store store.IStore
}

// 确保 biz 实现了 IBiz 接口。
var _ IBiz = (*biz)(nil)

// NewBiz 创建 IBiz 实例。
func NewBiz(store store.IStore) *biz {
	return &biz{store: store}
}

// UserV1 返回一个实现了 UserBiz 接口的实例.
func (b *biz) UserV1() userv1.UserBiz {
	return userv1.New(b.store, nil)
}
