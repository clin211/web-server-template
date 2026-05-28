# Go 内存模型要点

## 核心原则

Go 内存模型定义了在什么条件下，一个 goroutine 对变量的读取可以观测到另一个 goroutine 对同一变量的写入。

### Happens-Before 关系

如果事件 e1 发生在事件 e2 之前（happens-before），则 e2 应该能观测到 e1 的效果。

happens-before 是一个偏序关系，由以下规则推导：

## 同步保证

### 1. 包的初始化
- 如果包 p 导入了包 q，q 的 init 函数 happens-before p 的 init 函数
- main.main 函数 happens-before 所有 init 函数之后

### 2. Goroutine 创建
```go
go f()  // go 语句 happens-before f 的执行
```

### 3. Goroutine 销毁
goroutine 的退出不保证 happens-before 程序中任何事件。所以：
```go
var a string
go func() { a = "hello" }()
print(a)  // 可能打印空字符串！
```
需要用同步原语保证可见性。

### 4. Channel 通信
- 向 channel 的发送 happens-before 从该 channel 的接收完成
- 关闭 channel happens-before 从该 channel 接收到零值
- 无缓冲 channel 的接收 happens-before 发送完成
- 有缓冲 channel 的第 k 次接收 happens-before 第 k+C 次发送完成（C 为容量）

### 5. 锁
- `mu.Unlock()` happens-before 下一个 `mu.Lock()` 获取到锁
- `mu.RUnlock()` happens-before 下一个 `mu.Lock()` 或 `mu.RLock()`

### 6. Once
- `once.Do(f)` 中 f 的执行 happens-before 任何 `once.Do` 的返回

### 7. atomic
- atomic 操作提供了同步保证，类似于锁
- `atomic.Store` happens-before `atomic.Load` 读取到该值

## 数据竞争定义

满足以下所有条件即为数据竞争：
1. 两个 goroutine 访问同一个内存位置
2. 至少一个是写操作
3. 操作之间没有 happens-before 关系

## 常见内存模型陷阱

### 陷阱1：用共享变量通信
```go
// ❌ 数据竞争！
var done bool
go func() { done = true }()
for !done {}  // 可能永远循环（编译器优化）
```
**修复**: 用 channel 或 atomic。
```go
// ✅ channel
done := make(chan struct{})
go func() { close(done) }()
<-done
```

### 陷阱2：不正确的初始化检查
```go
// ❌ 可能看到部分初始化的值
var config *Config
go func() { config = loadConfig() }()
if config != nil {  // 无 happens-before 保证
    use(config)  // 可能使用未完全初始化的对象
}
```

### 陷阱3：编译器和 CPU 重排
没有同步原语保护的代码可能被重排：
```go
var a, b int
// goroutine 1
go func() { a = 1; b = 2 }()
// goroutine 2
go func() { print(b); print(a) }()  // 可能打印 "20"！
```
即使 g1 按顺序写入，g2 可能看到重排后的效果。

## 正确的同步模式

### 模式1：Mutex 保护共享变量
```go
var (
    mu sync.Mutex
    data map[string]int
)
func Write(k string, v int) {
    mu.Lock()
    data[k] = v  // Lock 保证 happens-before
    mu.Unlock()
}
func Read(k string) int {
    mu.Lock()
    defer mu.Unlock()
    return data[k]
}
```

### 模式2：Channel 传递所有权
```go
result := make(chan *Result)
go func() {
    r := compute()
    result <- r  // 发送 happens-before 接收
}()
r := <-result  // 接收者可以安全使用 r
```

### 模式3：atomic 保护简单状态
```go
var ready atomic.Bool
var config *Config

go func() {
    config = loadConfig()
    ready.Store(true)  // Store happens-before Load
}()

for !ready.Load() {}
use(config)  // 安全！
```

## race detector 使用

```bash
go test -race ./...           # 测试时检测
go build -race -o myapp       # 构建带检测的二进制
./myapp                        # 运行检测

# CI 中始终开启
# 产线环境不要用 -race（5-10x 慢，内存大增）
```