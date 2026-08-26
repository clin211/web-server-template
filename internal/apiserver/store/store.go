package store

import (
	"context"

	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// ProviderSet 是 Wire 提供者集，用于声明依赖注入规则。
// 它包含 NewStore 构造函数，用于生成 datastore 实例。
// wire.Bind 用于将 IStore 接口绑定到具体实现 *datastore，
// 允许在需要 IStore 的任何地方自动注入 *datastore 实例。
var ProviderSet = wire.NewSet(NewStore, wire.Bind(new(IStore), new(*datastore)))

// IStore 定义了 Store 层需要实现的方法。
type IStore interface {
	// DB 返回 Store 层的 *gorm.DB 实例，可能在少数情况下使用。
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
	// TX 用于在 Biz 层实现事务。回调内必须使用传入的 txCtx 调用其他 Store 方法；
	// 继续使用外层 ctx 会绕过事务连接，破坏业务数据与 Outbox/Inbox 状态的原子性。
	TX(ctx context.Context, fn func(ctx context.Context) error) error
	User() UserStore
	Outbox() OutboxStore
	Inbox() InboxStore
	DeadLetter() DeadLetterStore
	AuditLog() AuditLogStore
	UserLoginLog() UserLoginLogStore
}

// transactionKey 是用于在 context.Context 中存储事务上下文的键。
type transactionKey struct{}

// datastore 是 IStore 的具体实现。
type datastore struct {
	core *gorm.DB

	// 可以根据需要添加其他数据库实例。
	// 示例：fake *gorm.DB
}

// 确保 datastore 实现了 IStore 接口。
var _ IStore = (*datastore)(nil)

// NewStore 使用显式注入的数据库连接创建 datastore。
func NewStore(db *gorm.DB) *datastore {
	return &datastore{core: db}
}

// DB 根据输入条件（wheres）过滤数据库实例。
// 如果未提供条件，函数将从上下文返回数据库实例
// （事务实例或核心数据库实例）。
func (store *datastore) DB(ctx context.Context, wheres ...where.Where) *gorm.DB {
	db := store.core
	// 尝试从上下文中检索事务实例。
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		db = tx
	}

	// 将每个提供的 'where' 条件应用于查询。
	for _, whr := range wheres {
		db = whr.Where(db)
	}
	return db
}

// FakeDB 用于演示多个数据库实例。
// 它返回一个 nil 的 gorm.DB，表示一个假数据库。
func (ds *datastore) FakeDB(ctx context.Context) *gorm.DB { return nil }

// TX 启动一个新的事务实例。
//
// 事务连接通过 context 传播，使 Repository 保持小接口而不暴露 *gorm.DB。回调必须
// 继续传递这里提供的 context，DB(ctx) 才能选择事务连接而非进程级连接池。
// nolint: fatcontext
func (store *datastore) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	return store.core.WithContext(ctx).Transaction(
		func(tx *gorm.DB) error {
			ctx = context.WithValue(ctx, transactionKey{}, tx)
			return fn(ctx)
		},
	)
}

// User 返回一个实现了 UserStore 接口的实例.
func (store *datastore) User() UserStore {
	return newUserStore(store)
}

// Outbox 返回事务性 Outbox 存储。
func (store *datastore) Outbox() OutboxStore {
	return newOutboxStore(store)
}

// Inbox 返回幂等 Consumer Inbox 存储。
func (store *datastore) Inbox() InboxStore {
	return newInboxStore(store)
}

// DeadLetter 返回消费死信持久队列存储。
func (store *datastore) DeadLetter() DeadLetterStore {
	return newDeadLetterStore(store)
}

// AuditLog 返回审计日志存储。
func (store *datastore) AuditLog() AuditLogStore {
	return newAuditLogStore(store)
}

// UserLoginLog 返回登录安全审计存储。
func (store *datastore) UserLoginLog() UserLoginLogStore {
	return newUserLoginLogStore(store)
}
