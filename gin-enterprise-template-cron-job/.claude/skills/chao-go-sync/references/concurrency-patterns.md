# 并发模式

Go 语言中常见的并发"套路"——可解决某一类通用场景和问题的惯用法。

## 1. 半异步/半同步 Half-Async/Half-Sync

结合异步和同步两种并发模型的优点。

- **异步层**：处理高性能 I/O（如 epoll），单独的 goroutine 池
- **同步层**：提供简单的同步接口给调用者

**Go 中的应用**：
- Go 网络库：底层 epoll/kqueue 异步处理，上层提供 net.Conn 同步接口
- RPC 客户端：`client.Go`（异步）和 `client.Call`（同步，内部调用 Go + 等待 Done channel）

## 2. 活动对象 Active Object

解耦方法调用和方法执行，调用和执行在不同 goroutine 中。

### 组件

| 组件 | 说明 |
|------|------|
| Proxy | 客户端调用的接口，调用转为 method request |
| Method Request | 封装方法调用上下文 |
| Activation Queue | 待处理请求队列 |
| Scheduler | 独立 goroutine，调度方法执行 |
| Servant | 实际执行业务逻辑 |
| Future | 异步返回结果（Go 中用 channel 即可） |

### Go 实现（简化版）

```go
type Service struct {
    queue chan MethodRequest
    v     int
}
func (s *Service) schedule() {
    for r := range s.queue {
        if r == Incr { s.v++ } else { s.v-- }
    }
}
func (s *Service) Incr() { s.queue <- Incr }
```

## 3. 断路器 Circuit Breaker

防止故障扩散的故障保护机制。

### 三种状态

```
Closed ──失败达到阈值──→ Open ──超时后──→ Half-Open
  ↑                                        │
  └──────────成功达到阈值──────────────────┘
```

### 关键参数

- **MaxRequests**：半开状态允许的最大请求数
- **Interval**：闭合状态计数清空周期
- **Timeout**：断开状态持续时间
- **ReadyToTrip**：判断是否进入断开状态

## 4. 超时/截止时间 Deadline/Timeout

### Context 方式

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

### Timer 方式

```go
timer := time.NewTimer(maxTime)
defer timer.Stop()
select {
case <-timer.C: return // 超时
case result := <-workCh: process(result)
}
```

### 避免 time.After 内存泄漏

`time.After` 每次创建新 Timer，在过期前不释放。循环中应使用 `time.NewTimer` + `Reset`。

## 5. 回避模式 Balking

发现当前状态不适合执行时立即停止，不等待。

```go
var flag atomic.Bool
if !flag.CompareAndSwap(false, true) {
    return  // 已经有 goroutine 在执⾏，回避
}
defer flag.Store(false)
// 独占执行业务逻辑
```

## 6. 双检查 Double-Checked Locking

先无锁检查，不满足才加锁，加锁后再次检查。

**典型实现**：sync.Once

```go
if atomic.LoadUint32(&o.done) == 0 { // 第一次检查（无锁）
    o.doSlow(f)                       // 内部加锁后第二次检查
}
```

## 7. 保护式挂起 Guarded Suspension

条件不满足时挂起等待，条件满足后继续。

```go
func Guard(lock sync.Locker, fn func()) {
    lock.Lock()       // 条件满足（获取到锁）
    defer lock.Unlock()
    fn()
}
```

**channel 操作保护**（防止 panic）：

```go
func GuardClose[T any](ch chan T) {
    defer func() { recover() }()
    close(ch)
}
func GuardSend[T any](ch chan T, v T) {
    defer func() { recover() }()
    ch <- v
}
```

## 8. 核反应模式 Nuclear Reaction

多路数据的合并（核聚变）或分解（核裂变）。

- **聚变**：多个有序输入流合并为一个有序输出流
- **裂变**：一个大任务分解为多个子任务并行处理
- 例如：并发快排、控制 goroutine 数量的多级爬虫

## 9. 调度器模式 Scheduler

管理和调度多个并发任务。

**Go 中的应用**：GMP 调度器（Goroutine-Machine-Processor），包含 P 的调度、任务盗取、syscall 处理等。

**简化版 Dispatcher**：接收请求 → 分发给 Worker（可使用负载均衡算法）。

## 10. 反应器模式 Reactor

事件驱动 + 同步分发。用于处理大量并发 I/O。

- **主 Reactor**：负责 accept 连接
- **子 Reactor**：负责读写数据处理
- Go net 包本质是轻量级 Reactor：epoll 处理事件 → 交给 goroutine 处理

## 11. Proactor 模式

异步 I/O + 事件通知。I/O 操作完成后才触发回调。

Go 中较少使用（异步 I/O 非主流），[xtaci/gaio](https://github.com/xtaci/gaio) 是 Go 中的 Proactor 实现。

## 12. Per-CPU 模式

将数据分片到每个 P（逻辑处理器），同一 P 上同时只有一个 goroutine 运行 → 无数据竞争 → 无需锁。

- Go 标准库 sync.Pool 和 Timer 采用此设计
- [cespare/percpu](https://github.com/cespare/percpu)：高性能 per-CPU 计数器

## 13. 多进程模式

每个进程绑定一个 CPU 核，无数据竞争，利用 CPU 亲和性。

```go
// 通过 prefork 启动多个子进程，共享监听 socket
```

## 并发模式速查

| 场景 | 推荐模式 |
|------|---------|
| 网络服务高并发 | Half-Async/Half-Sync (Go 原生) |
| 方法异步执行 | Active Object |
| 防止故障扩散 | Circuit Breaker |
| 限制执行时间 | Deadline/Timeout |
| 防止重复执行 | Balking |
| 延迟初始化 | Double-Checked Locking |
| 等待条件满足 | Guarded Suspension |
| 多路数据合并 | Nuclear Reaction |
| 大量 I/O 处理 | Reactor / Proactor |
| 无锁高性能 | Per-CPU |