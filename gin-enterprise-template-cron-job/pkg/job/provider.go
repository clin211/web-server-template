// Package job 提供异步任务生产、定时调度、分布式锁和可观测性能力。
package job

import "github.com/google/wire"

// ProviderSet 声明任务模块依赖注入所需的 Wire provider 集合。
var ProviderSet = wire.NewSet(
	NewMetrics,
	NewAsynqProducer,
	NewScheduler,
	NewRedisLockWithClient,
	wire.Bind(new(Producer), new(*AsynqProducer)),
	wire.Bind(new(Locker), new(*RedisLock)),
)
