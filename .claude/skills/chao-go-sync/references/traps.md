# sync 原语常见陷阱与错误模式

## sync.Mutex 陷阱

### 1. 忘记 Unlock
最常见的错误，尤其在多分支路径中：
```go
// ❌ 错误：某些路径没有 Unlock
func (f *Foo) Bar() {
    f.mu.Lock()
    if f.count < 1000 {
        f.count += 3
        return  // 忘记 Unlock！
    }
    f.count++
    f.mu.Unlock()
}

// ✅ 正确：defer 保证释放
func (f *Foo) Bar() {
    f.mu.Lock()
    defer f.mu.Unlock()
    if f.count < 1000 {
        f.count += 3
        return
    }
    f.count++
}
```

### 2. 锁顺序不一致导致死锁
```go
// ❌ 错误：两个方法锁顺序相反
func (a *A) Method1() {
    a.mu.Lock(); defer a.mu.Unlock()
    b.mu.Lock(); defer b.mu.Unlock()  // A → B
}
func (b *B) Method2() {
    b.mu.Lock(); defer b.mu.Unlock()
    a.mu.Lock(); defer a.mu.Unlock()  // B → A  ← 死锁！
}
```
**解决**: 统一锁的获取顺序，或在设计层面避免嵌套锁。

### 3. 锁重入（递归锁）
Go Mutex 不支持可重入：
```go
// ❌ 死锁！
func (t *T) Foo() {
    t.mu.Lock(); defer t.mu.Unlock()
    t.Bar()
}
func (t *T) Bar() {
    t.mu.Lock(); defer t.mu.Unlock()  // 阻塞！
}
```

### 4. 复制 Mutex
```go
// ❌ go vet 可检测
var mu sync.Mutex
mu.Lock()
mu2 := mu  // 复制了锁的内部状态
mu.Unlock()
mu2.Lock()  // 不可预测的行为
```

### 5. 释放未持有的锁
```go
var mu sync.Mutex
mu.Unlock()  // panic: unlock of unlocked mutex
```

## sync.WaitGroup 陷阱

### 1. Add 放在 goroutine 内部
```go
// ❌ 错误
for i := 0; i < n; i++ {
    go func() {
        wg.Add(1)  // 可能与 wg.Wait() 竞态
        defer wg.Done()
        ...
    }()
}
wg.Wait()

// ✅ 正确
wg.Add(n)
for i := 0; i < n; i++ {
    go func() {
        defer wg.Done()
        ...
    }()
}
wg.Wait()
```

### 2. 计数器为负
Done 次数 > Add 次数会导致 panic：
```go
wg.Add(1)
wg.Done()
wg.Done()  // panic: negative WaitGroup counter
```

### 3. 重用 WaitGroup
```go
// ❌ 在 Wait() 返回前再次 Add
go func() {
    for {
        wg.Add(1)  // 并发调用，可能 panic
        wg.Done()
    }
}()
for { wg.Wait() }  // ④号 panic
```

### 4. 忘记 Add
```go
// ❌ 漏掉 Add，Wait 不会阻塞
go func() { defer wg.Done(); ... }()
wg.Wait()  // 立即返回！
```

## sync.Once 陷阱

### 1. 递归调用
```go
// ❌ 死锁
once.Do(func() {
    once.Do(func() { ... })  // 永远阻塞
})
```

### 2. panic 导致初始化失败
```go
once.Do(func() {
    conn, _ = net.Dial(...)  // 如果 panic，不会再初始化
})
// 后续调用者可能得到 nil conn
```

### 3. 初始化函数有返回值
Go 1.21 之前需要手动封装，现在用 OnceValue/OnceValues。

## sync.Pool 陷阱

### 1. 假设对象一定存在
```go
v := pool.Get()
// v 可能是 nil！
```

### 2. 假设对象状态
Put 回去的对象可能在 GC 时被回收，Get 到的对象状态是未定义的。

### 3. Pool 不适合 long-lived 对象
Pool 中的对象会在 GC 时释放，不适合需要持久化的缓存。

## sync.Map 陷阱

### 1. 不适合所有场景
- 不适合频繁写入和删除
- 不适合需要 Len() 的场景
- 需要 benchmark 验证收益

### 2. 类型安全
```go
// ❌ sync.Map 使用 interface{}，失去类型安全
m.Store("key", 42)
v, _ := m.Load("key")
i := v.(int)  // 需要类型断言

// ✅ Go 1.18+ 可以用泛型封装
type Map[K comparable, V any] struct { ... }
```

## channel 陷阱

### 1. 向已关闭的 channel 发送
```go
close(ch)
ch <- 1  // panic: send on closed channel
```

### 2. 关闭 nil channel
```go
var ch chan int
close(ch)  // panic: close of nil channel
```

### 3. 接收方关闭 channel
channel 应该由发送方关闭。

### 4. goroutine 泄漏
```go
ch := make(chan int)
go func() { ch <- 1 }()  // 如果没有接收者，goroutine 永远阻塞
```

## Context 陷阱

### 1. 存储在 struct 中
Context 应该作为函数参数传递：
```go
// ❌ 错误
type Server struct {
    ctx context.Context
}
// ✅ 正确
func (s *Server) Handle(ctx context.Context, ...) { ... }
```

### 2. 忘记检查 ctx.Done()
在长时间运行的操作中，应定期检查：
```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // 继续操作
}
```

## 通用陷阱

### 1. 在测试 goroutine 中使用 t.Fatal
```go
// ❌ 错误：t.Fatal 不能在非测试 goroutine 中调用
go func() {
    if err != nil { t.Fatal(err) }
}()

// ✅ 正确
go func() {
    if err != nil { t.Error(err); return }
}()
```

### 2. Lock 后复制
```go
mu.Lock()
copy := *sharedStruct  // 复制了锁保护的结构
mu.Unlock()
// copy 的内容可能不一致（部分更新）
```

### 3. 过度使用锁
每个操作都加锁会变成串行执行，失去并发优势。考虑：
- 减少临界区大小
- 使用分片锁
- 使用 lock-free 数据结构