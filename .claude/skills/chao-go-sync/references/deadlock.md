# 死锁诊断与 Goroutine 泄漏检测

## 死锁的经典条件（四个必要条件）

1. **互斥**: 资源不能共享，一次只能被一个 goroutine 使用
2. **持有并等待**: goroutine 持有至少一个资源，同时等待获取其他资源
3. **不可抢占**: 资源不能被强制释放，只能由持有者自愿释放
4. **循环等待**: 存在一个 goroutine 链，每个都在等待下一个持有的资源

四个条件**同时满足**才会死锁。破坏任意一个即可防止死锁。

## 定位死锁的方法

### 1. pprof goroutine profile
```bash
# 浏览器访问
http://localhost:8080/debug/pprof/goroutine?debug=1

# 或者命令行
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

观察点：
- 大量 goroutine 阻塞在同一个 Lock 调用
- 某个 goroutine 的栈显示 Lock 后没有对应的 Unlock

### 2. race detector
```bash
go test -race ./...
go build -race
```

虽然有性能代价（内存 5-10x，时间 2-20x），但在测试/CI 中开启非常重要。

### 3. Go 1.26+ 运行时泄漏检测
从 Go 1.26 起，运行时自动检测 goroutine 泄漏：
- 检测阻塞在 Mutex、Cond、channel 上且永远无法唤醒的 goroutine
- 基于 GC 可达性分析：如果同步原语 P 无法从任何可运行 goroutine 到达，则 P 上阻塞的 goroutine 是泄漏
- 无需特殊配置，运行时自动生效

### 4. 代码审查检查点
审查锁的获取和释放时，检查：
- [ ] 所有 Lock 是否有对应的 Unlock（包括所有分支/错误路径）
- [ ] 嵌套锁的获取顺序是否在所有代码路径中一致
- [ ] 是否存在可能的锁重入
- [ ] defer unlock 是否在被调用前正确设置

## 常见死锁模式

### 模式1：锁顺序死锁
```go
// goroutine 1: Lock(A) → Lock(B) → Unlock(B) → Unlock(A)
// goroutine 2: Lock(B) → Lock(A) → Unlock(A) → Unlock(B)
// 如果同时执行，g1 持有 A 等待 B，g2 持有 B 等待 A → 死锁
```

**修复**: 统一锁的获取顺序，或使用 `tryLock` 回退模式。

### 模式2：Channel 死锁
```go
ch := make(chan int)
ch <- 1  // 无缓冲 channel，没有接收者，永远阻塞
```

**修复**: 确保 channel 有接收者，或在 goroutine 中发送。

### 模式3：WaitGroup 死锁
```go
wg.Add(1)
// 忘记调用 Done()
wg.Wait()  // 永远等待
```

### 模式4：Once 递归死锁
```go
once.Do(func() {
    once.Do(func() { ... })  // 死锁
})
```

## Goroutine 泄漏

### 泄漏原因
1. channel 发送/接收没有对应的接收/发送方
2. goroutine 中的无限循环没有退出机制
3. select 中所有 case 都阻塞，没有 default 或退出路径

### 检测泄漏
```go
// 测试中检测 goroutine 泄漏
func TestNoLeak(t *testing.T) {
    before := runtime.NumGoroutine()
    // ... 执行被测代码
    time.Sleep(100 * time.Millisecond)  // 等待 goroutine 结束
    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("goroutine leak: %d → %d", before, after)
    }
}
```

推荐使用 `go.uber.org/goleak` 库在测试中自动检测。

### 防止泄漏的模式
```go
// ✅ 使用 Context 控制 goroutine 生命周期
func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return  // 正确退出
        case task := <-tasks:
            process(task)
        }
    }
}

// ✅ 使用 done channel 通知退出
func worker(done <-chan struct{}) {
    for {
        select {
        case <-done:
            return
        case task := <-tasks:
            process(task)
        }
    }
}
```

## 使用 synctest 测试并发代码（Go 1.25+）

```go
func TestConcurrent(t *testing.T) {
    synctest.Test(func(t testing.TB) {
        var mu sync.Mutex
        var counter int
        done := make(chan struct{})
        
        go func() {
            mu.Lock()
            counter++
            mu.Unlock()
            close(done)
        }()
        
        // synctest.Wait 等待所有 goroutine 阻塞
        // 时间虚拟化，不需要真的 sleep
        synctest.Wait()
    })
}
```