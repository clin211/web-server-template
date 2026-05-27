# 经典并发问题

## 1. 哲学家就餐问题

1971 年 Dijkstra 提出。五位哲学家围坐圆桌，需要两根筷子（左右各一）才能吃饭。

### 死锁条件（全部满足才会死锁）

1. **互斥**：筷子只能被一人持有
2. **持有并等待**：持有一根筷子，等待另一根
3. **禁止抢占**：不能抢夺别人的筷子
4. **循环等待**：每个人都持有一根并在等下一根

**破坏任意一个条件即可防止死锁。**

### 解法一：限制就餐人数（破坏循环等待）

最多允许 n-1 位同时就餐（信号量容量为 4）。

### 解法二：奇偶编号（破坏循环等待）

奇数先拿左手再拿右手，偶数先拿右手再拿左手：

```go
if p.ID%2 == 1 {
    p.leftChopstick.Lock()
    p.rightChopstick.Lock()
} else {
    p.rightChopstick.Lock()
    p.leftChopstick.Lock()
}
// ... 吃饭 ...
// 按相反顺序释放
```

### 解法三：资源分级（破坏循环等待）

筷子编号 1~5，所有哲学家先拿编号低的，再拿编号高的。释放时相反。最后一位哲学家（老子）先拿右手（编号1）再拿左手（编号5）。

### 解法四：引入服务生（破坏持有并等待）

拿两根筷子作为原子操作，外包给"服务生"（一把锁保护）：

```go
p.mu.Lock()  // 服务生控制
p.leftChopstick.Lock()
p.rightChopstick.Lock()
p.mu.Unlock()
// ... 吃饭 ...
```

---

## 2. 理发师问题

Dijkstra 1965 年提出。模拟多写单读的并发队列。

### 规则

- 无顾客 → 理发师睡觉
- 顾客到来 → 有空座就坐下，无空座就离开
- 理发师理发完 → 检查等待顾客，有则叫起，无则继续睡

### 解法一：sync.Cond 实现

```go
var seatsLock sync.Mutex
var seats int
var cond = sync.NewCond(&seatsLock)

// 理发师
seatsLock.Lock()
for seats == 0 {
    cond.Wait()  // 等待顾客
}
seats--
seatsLock.Unlock()

// 顾客
seatsLock.Lock()
if seats == capacity {
    return  // 离开
}
seats++
cond.Broadcast()  // 唤醒理发师
seatsLock.Unlock()
```

### 解法二：Channel 实现 Semaphore

```go
type Semaphore chan struct{}

func (s Semaphore) TryAcquire() bool {
    select {
    case s <- struct{}{}: return true
    default: return false
    }
}
func (s Semaphore) Release() { <-s }
```

### 注意事项

- 使用 Cond 时必须 for 循环检查条件（防止虚假唤醒）
- Channel 实现的 Semaphore 如有大量等待者，channel 容量会很大
- 可改进为 channel 中存顾客类型（而非空结构体），实现真正的顾客队列

### 多理发师扩展

多个理发师场景直接复用同一 Semaphore，天然支持多读多写。

---

## 3. 水工厂问题

两个氢原子 + 一个氧原子 → 一个水分子。必须三个 goroutine 都准备好才能生成。

### 解法：CyclicBarrier + Semaphore

```go
type H2O struct {
    semaH *semaphore.Weighted // 容量 2
    semaO *semaphore.Weighted // 容量 1
    b     cyclicbarrier.CyclicBarrier // parties=3
}

func (h2o *H2O) hydrogen(release func()) {
    h2o.semaH.Acquire(ctx, 1)  // 排队：保证两个 H goroutine
    release()                    // 输出 H
    h2o.b.Await(ctx)            // 等待三个原子齐备
    h2o.semaH.Release(1)        // 释放，准备下一轮
}

func (h2o *H2O) oxygen(release func()) {
    h2o.semaO.Acquire(ctx, 1)
    release()                    // 输出 O
    h2o.b.Await(ctx)
    h2o.semaO.Release(1)
}
```

### 简化版（无 CyclicBarrier）

如果不要求两个 H 来自不同 goroutine，只需两个信号量互相通知：

```go
// 氢: 获取 H 信号量 → 输出 H → 释放 O 信号量
// 氧: 获取 O 信号量(2个) → 输出 O → 释放 H 信号量(2个)
```

---

## 4. Fizz Buzz 问题

四个 goroutine 交替输出：3 的倍数 → "fizz"，5 的倍数 → "buzz"，15 的倍数 → "fizzbuzz"，其他 → 数字。

### 解法：将并发转为串行

四个 goroutine 通过 channel 链传递数字，处理不了就交给下一个：

```go
type FizzBuzz struct {
    chs [4]chan int  // goroutine 之间的通道
}
// fizz: 从 chs[0] 读，处理 3 的倍数，传给 chs[1]
// buzz: 从 chs[1] 读，处理 5 的倍数，传给 chs[2]
// fizzbuzz: 从 chs[2] 读，处理 15 的倍数，传给 chs[3]
// number: 从 chs[3] 读，处理其他数字，数字+1 传回 chs[0]
```

---

## 预防死锁策略总结

| 策略 | 破坏的条件 | 示例 |
|------|-----------|------|
| 限制并发数 | 循环等待 | 最多 4 人同时就餐 |
| 统一锁顺序 | 循环等待 | 奇偶编号、资源分级 |
| 原子化操作 | 持有并等待 | 服务生控制，一次拿两根 |