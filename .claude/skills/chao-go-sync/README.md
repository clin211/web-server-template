# chao-go-sync

Go 并发编程专家 Skill，基于《Go并发编程实战》一书，为 AI Coding Agent 提供专业的 Go 并发编程指导。

## 功能

- **并发 Bug 诊断** — 分析死锁、数据竞争、goroutine 泄漏、锁重入等问题
- **性能优化** — 锁粒度优化、分片方案、lock-free 替代、sync.Pool 优化
- **代码审查** — 检查 sync 原语的使用陷阱和错误模式
- **设计建议** — 为并发场景推荐合适的同步原语、并发模式和架构方案
- **版本迁移** — 建议使用新版本 Go 的 sync API 改进代码（Go 1.20 ~ 1.27）
- **分布式并发** — 基于 etcd 的选举、分布式锁、队列、屏障、STM 等原语建议

## 知识覆盖

### 标准库并发原语
Mutex、RWMutex、WaitGroup、Cond、Once、Pool、sync.Map、atomic、channel、context、synctest

### 官方扩展并发原语
信号量 (Semaphore)、SingleFlight、ErrGroup、限流 (Rate Limiter)

### 第三方并发库
CyclicBarrier、SizedGroup/ErrSizedGroup、gollback、Hunch、schedgroup、juju/ratelimit、uber-go/ratelimit、go-redis/redis_rate（分布式限流）、sony/gobreaker（断路器）、sourcegraph/conc、panjf2000/ants（Worker Pool）、cespare/percpu、valyala/bytebufferpool、cenk/backoff 等

### 并发模式 (13+ 种)
半异步半同步、活动对象、断路器、超时/截止时间、回避模式、双检查、保护式挂起、核反应、调度器、反应器、Proactor、Per-CPU、多进程

### 分布式同步原语 (基于 etcd)
Leader 选举、Locker/Mutex/RWMutex（分布式锁）、分布式队列/优先级队列、Barrier/DoubleBarrier（分布式屏障）、STM（软件事务内存）

### 经典并发问题
哲学家就餐（4 种解法）、理发师问题、水工厂问题、Fizz Buzz 问题

## 安装

```bash
npx skills add smallnest/chao-go-sync
```

或者在你的智能体对话框中输入：
```
安装skill: https://github.com/smallnest/chao-go-sync
```

安装后，在 Claude Code 或其他支持 Skills 的 AI Coding Agent 中，当讨论 Go 并发相关话题时自动激活，也可通过 `/chao-go-sync` 手动调用。

## 触发关键词

Go 并发、sync、Mutex、WaitGroup、Cond、Once、Pool、RWMutex、channel、goroutine、死锁、数据竞争、原子操作、sync.Map、并发优化、锁优化、Go concurrency、data race、deadlock、并发模式、并发设计、信号量、Semaphore、SingleFlight、ErrGroup、限流、令牌桶、漏桶、断路器、CyclicBarrier、分布式锁、etcd、选主、STM、Worker Pool

## 参考文件

本 Skill 包含以下参考资料：

### 基础
- `references/traps.md` — 各 sync 原语的常见陷阱和错误模式
- `references/patterns.md` — 并发设计模式和最佳实践
- `references/version-changes.md` — Go 1.20 ~ 1.27 sync 包变更详情
- `references/deadlock.md` — 死锁诊断和 goroutine 泄漏检测
- `references/memory-model.md` — Go 内存模型要点

### 扩展
- `references/extended-primitives.md` — 官方扩展并发原语（信号量、SingleFlight、ErrGroup、限流）
- `references/third-party-libs.md` — 第三方并发库（CyclicBarrier、断路器等）
- `references/concurrency-patterns.md` — 13+ 种并发模式速查
- `references/distributed-primitives.md` — 基于 etcd 的分布式同步原语
- `references/classic-problems.md` — 经典并发问题与解法

## 许可证

MIT License
