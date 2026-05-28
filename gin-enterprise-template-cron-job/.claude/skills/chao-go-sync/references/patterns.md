# 并发设计模式与最佳实践

## 1. 临界区保护模式

### 1.1 最小锁持有时间
```go
// ❌ 锁持有时间长
mu.Lock()
result := expensiveOperation()  // 不需要锁保护
data[result] = value
mu.Unlock()

// ✅ 只锁需要保护的部分
result := expensiveOperation()
mu.Lock()
data[result] = value
mu.Unlock()
```

### 1.2 读写分离
当读操作远多于写操作（通常 >90%）时使用 RWMutex：
```go
type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}
func (c *Cache) Get(k string) string {
    c.mu.RLock(); defer c.mu.RUnlock()
    return c.data[k]
}
func (c *Cache) Set(k, v string) {
    c.mu.Lock(); defer c.mu.Unlock()
    c.data[k] = v
}
```

### 1.3 分片锁
高并发写场景下，将一把锁拆分为多把锁：
```go
const shardCount = 32

type ShardedMap[K comparable, V any] struct {
    shards [shardCount]struct {
        mu sync.Mutex
        m  map[K]V
    }
}

func (s *ShardedMap[K, V]) getShard(key K) *... {
    h := hash(key) % shardCount
    return &s.shards[h]
}
```

## 2. 并发编排模式

### 2.1 等待所有任务完成（WaitGroup）
```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}
wg.Wait()
```

Go 1.25+ 简化写法：
```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Go(func() { process(item) })
}
wg.Wait()
```

### 2.2 扇出扇入（Fan-out/Fan-in）
多个 worker 处理任务，结果汇总到 channel：
```go
func FanIn[T any](inputs ...<-chan T) <-chan T {
    out := make(chan T)
    var wg sync.WaitGroup
    for _, ch := range inputs {
        wg.Add(1)
        go func(c <-chan T) {
            defer wg.Done()
            for v := range c { out <- v }
        }(ch)
    }
    go func() { wg.Wait(); close(out) }()
    return out
}
```

### 2.3 Pipeline 模式
多个阶段串联处理：
```go
func Pipeline(input <-chan int) <-chan int {
    stage1 := multiplyBy2(input)
    stage2 := addOne(stage1)
    return stage2
}
```

### 2.4 条件等待（Cond 模式）
```go
// 生产者-消费者队列
type Queue struct {
    cond    *sync.Cond
    items   []interface{}
    maxSize int
}

func (q *Queue) Put(item interface{}) {
    q.cond.L.Lock()
    for len(q.items) >= q.maxSize {
        q.cond.Wait()  // 等待空间
    }
    q.items = append(q.items, item)
    q.cond.L.Unlock()
    q.cond.Signal()  // 通知消费者
}

func (q *Queue) Get() interface{} {
    q.cond.L.Lock()
    for len(q.items) == 0 {
        q.cond.Wait()  // 等待元素
    }
    item := q.items[0]
    q.items = q.items[1:]
    q.cond.L.Unlock()
    q.cond.Signal()  // 通知生产者
    return item
}
```

### 2.5 Context 超时控制
```go
func fetchWithTimeout(ctx context.Context, url string) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    // ...
}
```

## 3. 单例模式

### 3.1 Once（无返回值）
```go
var (
    once   sync.Once
    logger *Logger
)

func GetLogger() *Logger {
    once.Do(func() { logger = NewLogger() })
    return logger
}
```

### 3.2 OnceValue（Go 1.21+，有返回值）
```go
var GetConfig = sync.OnceValue(func() *Config {
    return loadConfig()
})
```

### 3.3 OnceValues（Go 1.21+，有返回值+错误）
```go
var GetDB = sync.OnceValues(func() (*sql.DB, error) {
    return sql.Open("mysql", dsn)
})
```

## 4. 对象池模式

```go
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func process(data string) {
    buf := bufPool.Get().(*bytes.Buffer)
    buf.Reset()
    defer bufPool.Put(buf)
    buf.WriteString(data)
    // ... 使用 buf
}
```

## 5. 信号量模式（限流）

```go
// 用 buffered channel 实现信号量
sem := make(chan struct{}, maxConcurrency)

func limitedTask() {
    sem <- struct{}{}  // 获取许可
    defer func() { <-sem }()  // 释放许可
    // ... 执行任务
}
```

## 6. 或模式（竞速）

多个 channel 中任一就绪即处理：
```go
select {
case <-ch1:
    // ch1 就绪
case <-ch2:
    // ch2 就绪
case <-time.After(timeout):
    // 超时
}
```

## 7. 原子状态机

```go
const (
    stateIdle    int32 = 0
    stateRunning int32 = 1
    stateStopped int32 = 2
)

var state atomic.Int32

func Start() bool {
    return state.CompareAndSwap(stateIdle, stateRunning)
}

func Stop() {
    state.Store(stateStopped)
}
```

Go 1.23+ 位标志操作：
```go
var flags atomic.Uint32
const (
    flagReady  = 1 << 0
    flagActive = 1 << 1
)
flags.Or(flagReady)          // 设置标志
flags.And(^flagActive)       // 清除标志
if flags.Load()&flagReady != 0 { ... }  // 检查标志
```

## 8. 并发安全 Map 的选型

| 场景 | 推荐方案 |
|------|---------|
| 少量 key，读写均衡 | map + RWMutex |
| key 写入一次，频繁读取 | sync.Map |
| 大量 key，高并发写入 | 分片 map（如 orcaman/concurrent-map） |
| 极端读性能要求 | lock-free map（如 cornelk/hashmap） |
| 需要 Len()/Range | 用 RWMutex 或分片方案 |

## 9. 并发测试模式

### Go 1.25+ synctest
```go
func TestTimeout(t *testing.T) {
    synctest.Test(func(t testing.TB) {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        go func() {
            time.Sleep(10 * time.Second)  // 虚拟时间，瞬间完成
        }()
        
        synctest.Wait()
        // ctx 应该已超时
    })
}
```

### 压力测试
```go
func TestConcurrentSafety(t *testing.T) {
    var wg sync.WaitGroup
    m := NewSafeMap()
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            for j := 0; j < 1000; j++ {
                m.Set(fmt.Sprintf("key-%d-%d", n, j), j)
            }
        }(i)
    }
    wg.Wait()
}
```

## 10. errgroup 模式

需要 "一组 goroutine 任一失败则全部取消" 时：
```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(ctx)
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