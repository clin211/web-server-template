# 分布式同步原语

基于 etcd 实现的分布式并发控制原语。

## 1. Leader 选举

包: `go.etcd.io/etcd/client/v3/concurrency`

### API

```go
func NewElection(s *Session, name string) *Election
func (e *Election) Campaign(ctx context.Context, val string) error  // 竞选
func (e *Election) Proclaim(ctx context.Context, val string) error  // 宣告新值（不重新选主）
func (e *Election) Resign(ctx context.Context) error                // 逊位
func (e *Election) Leader(ctx context.Context) (*v3.GetResponse, error)  // 查询当前主
func (e *Election) Observe(ctx context.Context) <-chan v3.GetResponse   // 监控主变化
```

### 使用流程

1. Campaign 阻塞调用，直到当选为主（或 ctx 取消/出错）
2. 从节点通过 Leader 查询当前主信息
3. 通过 Observe 获取一个 channel，监听主节点变动
4. 主节点可调用 Resign 辞去主角色

---

## 2. 分布式锁

### 2.1 Locker (简单分布式锁)

```go
func NewLocker(s *Session, pfx string) sync.Locker
```

实现了标准库 `sync.Locker` 接口（Lock/Unlock），使用最简单。

### 2.2 Mutex (带 TTL 的分布式锁)

```go
func NewMutex(s *Session, pfx string) *Mutex
func (m *Mutex) Lock(ctx context.Context) error
func (m *Mutex) Unlock(ctx context.Context) error
func (m *Mutex) Key() string
```

**核心特性**：持有锁的节点崩溃后，锁会在 TTL 后自动释放。

```go
// 设置 TTL 为 30 秒（默认 60 秒）
s, err := concurrency.NewSession(cli, concurrency.WithTTL(30))
m := concurrency.NewMutex(s, "my-lock")
m.Lock(ctx)
// ... 临界区
m.Unlock(ctx)
```

### 2.3 RWMutex (分布式读写锁)

包: `go.etcd.io/etcd/client/v3/experimental/recipes`

```go
func NewRWMutex(s *Session, key string) *RWMutex
func (rwm *RWMutex) Lock() error
func (rwm *RWMutex) Unlock() error
func (rwm *RWMutex) RLock() error
func (rwm *RWMutex) RUnlock() error
```

### 读写锁等待顺序

- 写锁被持有 → 读锁和写锁都等待
- 读锁被持有 → 写锁等待, 读锁直接获取
- 读锁被持有 + 有写锁等待 → 后续读锁也被阻塞（防止写饥饿，与标准库一致）

---

## 3. 分布式队列

包: `go.etcd.io/etcd/client/v3/experimental/recipes`

### Queue

```go
func NewQueue(client *Client, keyPrefix string) *Queue
func (q *Queue) Enqueue(val string) error
func (q *Queue) Dequeue() (string, error)
```

多读多写，空队列 Dequeue 会阻塞。可在不同节点入队/出队。

### PriorityQueue

```go
func NewPriorityQueue(client *Client, keyPrefix string) *PriorityQueue
func (pq *PriorityQueue) Enqueue(val string, prio uint16) error
func (pq *PriorityQueue) Dequeue() (string, error)
```

优先级数值越小越优先出队。

---

## 4. 分布式屏障

### 4.1 Barrier (一次性屏障)

```go
func NewBarrier(client *Client, key string) *Barrier
func (b *Barrier) Hold() error     // 创建屏障
func (b *Barrier) Release() error  // 释放屏障，所有等待者放行
func (b *Barrier) Wait() error     // 等待屏障释放
```

创建和释放可以由不同节点完成。

### 4.2 DoubleBarrier (两阶段屏障)

```go
func NewDoubleBarrier(s *Session, key string, count int) *DoubleBarrier
func (b *DoubleBarrier) Enter() error   // 等 count 个节点都 Enter 才放行
func (b *DoubleBarrier) Leave() error   // 等 count 个节点都 Leave 才放行
```

适用场景：编排一组分布式节点在相同时间点开始/结束执行。

---

## 5. STM (软件事务内存)

包: `go.etcd.io/etcd/client/v3/concurrency`

基于 CAS 的事务封装，一组读写操作原子执行。

```go
type STM interface {
    Get(key ...string) string
    Put(key, val string, opts ...v3.OpOption)
    Rev(key string) int64
    Del(key string)
}

func NewSTM(cli *Client, apply func(STM) error) (*v3.TxnResponse, error)
```

### 使用示例

```go
exchange := func(stm concurrency.STM) error {
    fromV := stm.Get(fromK)
    toV := stm.Get(toK)
    stm.Put(fromK, fmt.Sprintf("%d", fromInt - amount))
    stm.Put(toK, fmt.Sprintf("%d", toInt + amount))
    return nil
}
concurrency.NewSTM(cli, exchange)
```

事务保证多个 key 的更改要么全成功，要么全失败。

---

## 分布式同步原语速查

| 需求 | 原语 | 包 |
|------|------|----|
| 选主 | Election | concurrency |
| 简单分布式锁 | Locker | concurrency |
| 带 TTL 分布式锁 | Mutex | concurrency |
| 分布式读写锁 | RWMutex | recipes |
| 分布式队列 | Queue/PriorityQueue | recipes |
| 分布式屏障 | Barrier/DoubleBarrier | recipes |
| 分布式事务 | STM | concurrency |