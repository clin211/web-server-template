# Go 官方扩展并发原语

## 1. 信号量 Semaphore

包: `golang.org/x/sync/semaphore`

带权重的信号量实现，允许一次请求/释放多个资源。

### API

```go
func NewWeighted(n int64) *Weighted
func (s *Weighted) Acquire(ctx context.Context, n int64) error
func (s *Weighted) Release(n int64)
func (s *Weighted) TryAcquire(n int64) bool
```

### 内部实现

使用 Mutex + List 实现。size 为资源总数，cur 为已占用数，waiters 为等待者链表。每个 waiter 有一个 ready channel 用于唤醒通知。

`notifyWaiters` 遇到第一个不满足的 waiter 就停止，避免饥饿（如读写锁场景，写锁需要全部资源）。

### 使用示例

```go
sema := semaphore.NewWeighted(int64(maxWorkers))
for i := range tasks {
    if err := sema.Acquire(ctx, 1); err != nil {
        break
    }
    go func(i int) {
        defer sema.Release(1)
        process(tasks[i])
    }(i)
}
// 等待所有 worker 完成
sema.Acquire(ctx, int64(maxWorkers))
```

### 常见错误

- 请求了资源但忘记释放
- Release 传递比 Acquire 更大的值 → panic
- 请求资源数超过最大资源数 → 永久阻塞
- 传递负数给 Release → 资源永久被持有

### 使用场景

- Worker Pool 并发控制
- 数据库连接池限制
- containerd 等项目中用于限流控制

---

## 2. SingleFlight

包: `golang.org/x/sync/singleflight`

合并并发请求，对同一 key 的并发调用只执行一次，结果共享。

### 与 sync.Once 的区别

| 特性 | sync.Once | SingleFlight |
|------|-----------|-------------|
| 执行次数 | 永远只执行一次 | 每次调用重新执行 |
| 并发行为 | 一个执行，其他等待 | 合并同时的请求 |
| 场景 | 单次初始化 | 缓存击穿防护 |

### API

```go
type Group struct { ... }
func (g *Group) Do(key string, fn func() (interface{}, error)) (v interface{}, err error, shared bool)
func (g *Group) DoChan(key string, fn func() (interface{}, error)) <-chan Result
func (g *Group) Forget(key string)
```

### 内部实现

Mutex + Map[string]*call。call 包含 WaitGroup 用于等待第一个请求完成。

### 使用场景

- **缓存击穿防护**：groupcache、CockroachDB、CoreDNS 都使用了 SingleFlight
- **合并 DNS 查询**：Go 标准库 `net/lookup.go` 中用于合并同一 host 的查询
- **秒杀场景**：大并发读请求合并为一个

```go
var g singleflight.Group
func getData(key string) (interface{}, error) {
    return g.Do(key, func() (interface{}, error) {
        return fetchFromDB(key)
    })
}
```

### 注意事项

- SingleFlight 合并的是并发请求，不能合并并发的写操作
- Forget 可以提前"忘记"key，让后续请求发起新调用
- shared 返回值指示结果是否被多个调用者共享

---

## 3. ErrGroup

包: `golang.org/x/sync/errgroup`

WaitGroup 的增强版，集成 Context 和 error 传播。

### API

```go
func WithContext(ctx context.Context) (*Group, context.Context)
func (g *Group) Go(f func() error)
func (g *Group) TryGo(f func() error) bool
func (g *Group) Wait() error
func (g *Group) SetLimit(n int)
```

### 特性

- **error 传播**：第一个非 nil error 会取消 Context，Wait 返回该 error
- **Context 集成**：WithContext 返回派生 Context，在首错或 Wait 返回时取消
- **并发限制**：SetLimit 限制同时活跃的 goroutine 数
- **TryGo**：非阻塞启动，超过 limit 则返回 false

### 内部实现

WaitGroup + 信号量(channel) + Once + Context 组合。

### 使用示例

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(20) // 最多 20 个并发

for _, url := range urls {
    url := url
    g.Go(func() error {
        return fetch(ctx, url)
    })
}
if err := g.Wait(); err != nil {
    log.Fatal(err)
}
```

### 收集所有子任务结果

ErrGroup 只返回第一个错误，如需收集全部结果，使用额外 slice：

```go
var results = make([]Result, len(tasks))
g.Go(func() error {
    results[i] = process(tasks[i])
    return results[i].Err
})
```

---

## 4. 限流 Rate Limiting

### 4.1 令牌桶: x/time/rate

包: `golang.org/x/time/rate`

```go
func NewLimiter(r Limit, b int) *Limiter
func (lim *Limiter) Allow() bool     // 非阻塞
func (lim *Limiter) Reserve() *Reservation  // 保留未来令牌
func (lim *Limiter) Wait(ctx context.Context) error  // 阻塞等待
```

桶容量 = b，初始满桶，以速率 r 填充。允许最大 b 个突发请求。

`SetLimitAt(t, newLimit)` 中的 t 含义：t 减去最后生成令牌时间得到 elapsed，按原 limit 计算令牌放入桶中（不超过 burst），然后设置新 Limit。**令牌立即按新速率生成**，不是等到 t 时刻。

### 4.2 令牌桶 vs 漏桶

| 特性 | 令牌桶 | 漏桶 |
|------|--------|------|
| 输出速率 | 平均恒定，允许突发 | 严格恒定 |
| 突发处理 | 允许 burst | 削峰填谷，平滑处理 |
| 适用场景 | 需要处理突发流量 | 需要严格平滑输出 |
| 实现 | x/time/rate, juju/ratelimit | uber-go/ratelimit |

### 4.3 使用建议

- 优先使用 Wait 方法，避免非阻塞方法的复杂设计
- 如果不想初始满桶，创建后立即 Take/Wait 掉容量大小的令牌
- 漏桶 + WithSlack 可以兼顾突发支持