# 第三方并发库

## 1. 循环屏障 CyclicBarrier

包: `github.com/marusama/cyclicbarrier`

可循环使用的屏障，允许一组 goroutine 互相等待到达同一执行点。

### 与 WaitGroup 的区别

| 特性 | WaitGroup | CyclicBarrier |
|------|-----------|---------------|
| 等待方向 | 父等子 | 参与者互相等待 |
| 可重用 | 不擅长（需重置计数） | 天然可重用 |
| 适用场景 | "一个等一组" | "固定数量互相等待，多轮" |

### API

```go
func New(parties int) CyclicBarrier
func NewWithAction(parties int, barrierAction func() error) CyclicBarrier

type CyclicBarrier interface {
    Await(ctx context.Context) error
    Reset()
    GetNumberWaiting() int
    GetParties() int
    IsBroken() bool
}
```

### happens-before 保证

任意 goroutine 的第 n 次 Await 调用，一定 synchronized-before 任意 goroutine 的第 n+1 次 Await 成功返回。

### 使用示例

```go
b := cyclicbarrier.NewWithAction(10, func() error {
    cnt++  // 每轮屏障放开时执行
    return nil
})
for i := 0; i < 10; i++ {
    go func() {
        for j := 0; j < 5; j++ {
            time.Sleep(randomDuration())
            b.Await(context.TODO())  // 等待所有参与者
        }
    }()
}
```

---

## 2. 分组操作库

### 2.1 SizedGroup / ErrSizedGroup

包: `github.com/go-pkgz/syncs`

信号量 + WaitGroup 实现。控制并发 goroutine 数量执行大量子任务。

```go
swg := syncs.NewSizedGroup(10)           // 默认：控制子任务并发数
swg := syncs.NewSizedGroup(10, syncs.Preemptive) // 控制 goroutine 数
for i := 0; i < 1000; i++ {
    swg.Go(func(ctx context.Context) { process() })
}
swg.Wait()
```

ErrSizedGroup 额外提供 error 处理和 termOnError 模式。

### 2.2 gollback

包: `github.com/vardius/gollback`

解决 ErrGroup 收集结果痛点，直接返回结果和错误。

```go
// All: 等待所有完成，返回全部结果
rs, errs := gollback.All(ctx, fn1, fn2, fn3)

// Race: 任一成功即返回，同时 cancel 其他
result, err := gollback.Race(ctx, fn1, fn2, fn3)

// Retry: 重试机制
result, err := gollback.Retry(ctx, 5, fn)
```

### 2.3 Hunch

包: `github.com/AaronJan/Hunch`

功能比 gollback 更丰富：

| 方法 | 说明 |
|------|------|
| `All(ctx, execs...)` | 任一错误立即返回 |
| `Take(ctx, num, execs...)` | 取前 num 个成功结果 |
| `Last(ctx, num, execs...)` | 取最后 num 个成功结果 |
| `Retry(ctx, retries, fn)` | 重试 |
| `Waterfall(ctx, execs...)` | 串行 pipeline，前一个结果传给下一个 |

### 2.4 schedgroup

包: `github.com/mdlayher/schedgroup`

定时任务分组，使用 container/heap 排序执行时间避免大量 timer。

```go
sg := schedgroup.New(ctx)
sg.Delay(100*time.Millisecond, fn1)  // 100ms 后执行
sg.Schedule(time.Now().Add(1*time.Second), fn2)  // 指定时间执行
sg.Wait()  // 等待所有任务完成
```

注意：Wait 后不能再调用 Delay/Schedule，Wait 只能调用一次。

---

## 3. 限流库

### 3.1 juju/ratelimit (令牌桶)

包: `github.com/juju/ratelimit`

亮点：quantum 参数可在每次生成令牌时产生多个。

```go
bucket := ratelimit.NewBucket(time.Second, 3)  // 每秒1个，容量3
bucket.Wait(1)          // 阻塞等待
bucket.TakeAvailable(5) // 能拿多少拿多少
```

还提供 `Reader/Writer` 方法实现 I/O 限流（每个令牌 = 1 byte）。

### 3.2 uber-go/ratelimit (漏桶)

包: `go.uber.org/ratelimit`

极其简洁：只有一个 Take 方法。

```go
rl := ratelimit.New(100)                          // 每秒100个
rl := ratelimit.New(100, ratelimit.WithSlack(3))  // 允许积攒3个突发
rl.Take()  // 阻塞获取
```

### 3.3 分布式限流: go-redis/redis_rate

包: `github.com/go-redis/redis_rate`

基于 Redis + Lua 脚本的漏桶实现。

```go
limiter := redis_rate.NewLimiter(rdb)
res, _ := limiter.Allow(ctx, "token:123", redis_rate.PerSecond(5))
// res.Allowed, res.Remaining, res.RetryAfter
```

同一 key 跨节点共享配额，实现分布式限流。

---

## 4. 断路器 Circuit Breaker

### 4.1 sony/gobreaker

包: `github.com/sony/gobreaker`

三种状态：Closed → Open → Half-Open → Closed

```go
var st gobreaker.Settings
st.Name = "HTTP GET"
st.ReadyToTrip = func(counts gobreaker.Counts) bool {
    failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
    return counts.Requests >= 3 && failureRatio >= 0.6
}
cb := gobreaker.NewCircuitBreaker(st)

body, err := cb.Execute(func() (interface{}, error) {
    return doRequest(url)
})
```

### 4.2 mercari/go-circuitbreaker

包: `github.com/mercari/go-circuitbreaker`

功能类似，API 使用 Optional Function Parameter Pattern。

---

## 5. Worker Pool 库

| 库 | 特点 |
|----|------|
| `gammazero/workerpool` | 无限制提交, Submit/SubmitWait |
| `ivpusic/grpool` | 固定 worker 数 + 最大任务数 |
| `dpaks/goworkers` | ResultChan/ErrChan 获取结果 |
| `panjf2000/ants` | 高性能 goroutine 池，自动管理 |
| `Jeffail/tunny` | 支持多 pool 类型 |
| `alitto/pond` | 任务组、错误处理 |

### 评估标准

1. 功能是否满足需求
2. star 数（用户基数）
3. 代码活跃度（维护状态）
4. 代码可读性
5. 代码简洁性

---

## 7. Buffer 池库

### bytebufferpool

包: `github.com/valyala/bytebufferpool`

fasthttp 作者开发。亮点：校准（calibrate）机制，动态调整 defaultSize/maxSize，智能偏向常用 size 范围。

### bpool

包: `github.com/oxtoacart/bpool`

Channel 实现，特色：限制池容量，超过阈值自动丢弃。

| 类型 | 说明 |
|------|------|
| BufferPool | 固定数量 bytes.Buffer 池 |
| BytesPool | 固定数量 byte slice 池 |
| SizedBufferPool | 带 size 检查的 buffer 池 |

---

## 8. 其他实用库

| 库 | 用途 |
|----|------|
| `sourcegraph/conc` | 更好的结构化同步原语，panic 保护 |
| `cenk/backoff` | 指数退避重试 |
| `sethvargo/go-retry` | 可配置重试策略 |
| `cespare/percpu` | Per-CPU 计数器，避免 false sharing |
| `petermattis/goid` / `kortschak/goroutine` | 获取 goroutine ID（用于可重入锁等） |