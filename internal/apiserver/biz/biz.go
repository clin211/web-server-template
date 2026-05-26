package biz

import (
	menuv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/menu"
	permissionv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/permission"
	rolev1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/role"
	scheduledtaskv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/scheduled_task"
	userv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/user"
	userrolev1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/user_role"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/pkg/authz"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
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
	// RoleV1 获取角色业务接口.
	RoleV1() rolev1.RoleBiz
	// PermissionV1 获取权限业务接口.
	PermissionV1() permissionv1.PermissionBiz
	// MenuV1 获取菜单业务接口.
	MenuV1() menuv1.MenuBiz
	// UserRoleV1 获取用户角色业务接口.
	UserRoleV1() userrolev1.UserRoleBiz
	// ScheduledTaskV1 获取定时任务业务接口.
	ScheduledTaskV1() scheduledtaskv1.ScheduledTaskBiz
}

// biz 是 IBiz 的具体实现。
type biz struct {
	store      store.IStore
	authz      *authz.Authz
	producer   *genericjob.AsynqProducer
	scheduler  *genericjob.Scheduler
	registry   *genericjob.Registry
	jobOptions *genericoptions.JobOptions
}

// 确保 biz 实现了 IBiz 接口。
var _ IBiz = (*biz)(nil)

// NewBiz 创建 IBiz 实例。
func NewBiz(store store.IStore, authz *authz.Authz, producer *genericjob.AsynqProducer, scheduler *genericjob.Scheduler, registry *genericjob.Registry, jobOptions *genericoptions.JobOptions) *biz {
	return &biz{store: store, authz: authz, producer: producer, scheduler: scheduler, registry: registry, jobOptions: jobOptions}
}

// UserV1 返回一个实现了 UserBiz 接口的实例.
func (b *biz) UserV1() userv1.UserBiz {
	return userv1.New(b.store, b.authz)
}

// RoleV1 返回一个实现了 RoleBiz 接口的实例.
func (b *biz) RoleV1() rolev1.RoleBiz {
	return rolev1.New(b.store, b.authz)
}

// PermissionV1 返回一个实现了 PermissionBiz 接口的实例.
func (b *biz) PermissionV1() permissionv1.PermissionBiz {
	return permissionv1.New(b.store)
}

// MenuV1 返回一个实现了 MenuBiz 接口的实例.
func (b *biz) MenuV1() menuv1.MenuBiz {
	return menuv1.New(b.store)
}

// UserRoleV1 返回一个实现了 UserRoleBiz 接口的实例.
func (b *biz) UserRoleV1() userrolev1.UserRoleBiz {
	return userrolev1.New(b.store, b.authz)
}

// ScheduledTaskV1 返回一个实现了 ScheduledTaskBiz 接口的实例.
func (b *biz) ScheduledTaskV1() scheduledtaskv1.ScheduledTaskBiz {
	return scheduledtaskv1.New(b.store, b.authz, b.producer, b.scheduler, b.registry, b.jobOptions)
}
