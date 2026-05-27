# Go 1.20 → 1.27 sync 包变更详情

## Go 1.21 (2023-08)

### sync 包新增
| API | 说明 |
|-----|------|
| `sync.OnceFunc(f func()) func()` | 返回只执行一次的函数 |
| `sync.OnceValue[T any](f func() T) func() T` | 返回只执行一次并返回 T 的函数 |
| `sync.OnceValues[T1, T2 any](f func() (T1, T2)) func() (T1, T2)` | 返回只执行一次并返回两个值的函数 |

### context 包新增
| API | 说明 |
|-----|------|
| `context.WithTimeoutCause(parent, timeout, cause)` | 超时并携带取消原因 |
| `context.WithDeadlineCause(parent, d, cause)` | 截止时间并携带取消原因 |
| `context.AfterFunc(ctx, f) (stop func() bool)` | ctx 完成后执行 f |
| `context.Cause(ctx) error` | 获取取消原因 |

### 迁移建议
```go
// 旧写法
var once sync.Once
var cfg *Config
func GetConfig() *Config {
    once.Do(func() { cfg = loadConfig() })
    return cfg
}
// 新写法
var GetConfig = sync.OnceValue(func() *Config { return loadConfig() })
```

## Go 1.22 (2024-02)

sync 包无新增 API。

### 运行时变更
- mutex profile 变更：`/sync/mutex/wait/total:seconds` 现在包含 runtime 内部锁的竞争
- for-range 循环变量语义变更（每个迭代的变量独立，不再需要 `x := x`）

## Go 1.23 (2024-08)

### sync 包新增
| API | 说明 |
|-----|------|
| `sync.Map.Clear()` | 清空所有条目 |

### sync/atomic 包新增
| API | 说明 |
|-----|------|
| `AndInt32/Int64/Uint32/Uint64/Uintptr(addr, mask)` | 原子按位 AND |
| `OrInt32/Int64/Uint32/Uint64/Uintptr(addr, mask)` | 原子按位 OR |
| 对应类型的 `And()`/`Or()` 方法 | atomic.Int32 等类型的方法 |

### 迁移建议
```go
// 旧写法：CAS 循环清除标志位
for {
    old := flags.Load()
    if flags.CompareAndSwap(old, old &^ flagActive) { break }
}
// 新写法
flags.And(^flagActive)
```

## Go 1.24 (2025-02)

### sync 包
| 变更 | 说明 |
|------|------|
| sync.Map 实现重写 | hash-trie map，性能大幅提升 |
| GOEXPERIMENT=nosynchashtriemap | 回退到旧实现的环境变量 |

### sync.Map 新增方法
| API | 说明 |
|-----|------|
| `Map.CompareAndSwap(key, old, new) bool` | 原子 CAS 更新 |
| `Map.CompareAndDelete(key, old) (deleted bool)` | 原子比较并删除 |

### testing/synctest (GOEXPERIMENT=synctest)
实验性包，用于在隔离的时间环境中测试并发代码。
- `synctest.Run`: 在隔离的 "气泡" 中启动 goroutine
- `synctest.Wait`: 等待所有 goroutine 阻塞

## Go 1.25 (2025-08)

### sync 包新增
| API | 说明 |
|-----|------|
| `sync.WaitGroup.Go(f func())` | Add(1) + goroutine + Done 的一体化方法 |

### testing/synctest (GA)
synctest 从实验性转为正式 API：
- `synctest.Test(f)`: 替代 synctest.Run
- `synctest.Wait`: 等待所有 goroutine 阻塞
- 时间虚拟化，所有 goroutine 阻塞时时钟瞬间推进

```go
// 迁移建议
var wg sync.WaitGroup
for _, item := range items {
    wg.Go(func() { process(item) })  // Go 1.25+
}
```

## Go 1.26 (2026-02)

### 运行时
| 变更 | 说明 |
|------|------|
| goroutine 泄漏检测 | 自动检测阻塞在 sync.Mutex、sync.Cond、channel 等上永远无法唤醒的 goroutine |
| 利用 GC 检测 | 如果 goroutine G 阻塞在同步原语 P，且 P 无法被任何可运行 goroutine 到达，则报告泄漏 |

sync 包无新增 API。

## Go 1.27 (2026-08, 预期)

### testing/synctest
| API | 说明 |
|-----|------|
| `synctest.Sleep(d)` | 组合 time.Sleep + synctest.Wait |

### 清理
- 旧 `GOEXPERIMENT=synctest` API 移除
- `asynctimerchan` GODEBUG 移除

sync 包本身无重要 API 变更。

## 版本兼容速查

| 特性 | 最低 Go 版本 |
|------|-------------|
| TryLock / TryRLock | 1.18 |
| atomic.Int32 等类型安全原子类型 | 1.19 |
| context.WithCancelCause | 1.20 |
| OnceFunc / OnceValue / OnceValues | 1.21 |
| context.WithTimeoutCause / AfterFunc | 1.21 |
| sync.Map.Clear | 1.23 |
| atomic.And / atomic.Or | 1.23 |
| sync.Map.CompareAndSwap / CompareAndDelete | 1.24 |
| sync.WaitGroup.Go | 1.25 |
| testing/synctest (GA) | 1.25 |
| goroutine 泄漏检测 | 1.26 |