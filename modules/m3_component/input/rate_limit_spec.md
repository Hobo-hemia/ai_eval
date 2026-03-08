# M3 组件设计题：承受雷群打击（Thundering Herd）的高并发本地缓存

## 场景背景

你需要实现一个本地缓存中间件，它将被嵌入到日均百亿请求的网关微服务中。
该组件的核心作用是拦截对底层极其脆弱的 RPC 服务的访问（例如风控规则加载、大鉴权计算）。

## 致命的环境约束

在过去的一周，我们的线上服务因为这个组件的旧版本发生了 3 次 P0 级宕机。原因如下：

1. **缓存击穿与惊群效应（Thundering Herd）**：当某个热点 key（如“双十一全局配置”）突然过期时，在1毫秒内会有超过 10,000 个 goroutine 同时发现缓存 miss，并同时调用 `LoaderFunc` 去查询底层 RPC。底层 RPC 瞬间被打挂。
2. **底层服务“假死”引发的雪崩**：`LoaderFunc` 所依赖的外部服务有时会陷入长达一分钟的假死（不返回错误，一直 hang 住）。旧版缓存导致所有请求 goroutine 堆积在等待响应上，耗尽了网关的内存和协程池，引发连锁雪崩。
3. **资源限制与垃圾堆积**：网关可用内存极其受限。曾经发生过因为大量“长尾无用 key”持续吃内存，导致应用被容器平台的 OOM Killer 干掉。我们必须在不暂停整个世界（Stop The World）的前提下，实现平滑驱逐。

## 必须实现的对外契约（不可改签名）

```go
type CacheConfig struct {
    ShardCount         int
    MaxEntriesPerShard int
    DefaultTTL         time.Duration
    CleanupInterval    time.Duration
}

type CacheStats struct {
    Hits         uint64
    Misses       uint64
    LoadSuccess  uint64
    LoadFailures uint64
    Evictions    uint64
}

// 警告：loader 可能极其缓慢甚至 hang 住
type LoaderFunc[K comparable, V any] func(ctx context.Context, key K) (value V, ttl time.Duration, err error)

type ShardCache[K comparable, V any] struct { /* unexported fields */ }

func NewShardCache[K comparable, V any](cfg CacheConfig) (*ShardCache[K, V], error)
func (c *ShardCache[K, V]) Get(ctx context.Context, key K) (V, bool)
func (c *ShardCache[K, V]) Set(ctx context.Context, key K, value V, ttl time.Duration)
func (c *ShardCache[K, V]) Delete(ctx context.Context, key K)
// 核心难点方法：
func (c *ShardCache[K, V]) GetOrLoad(ctx context.Context, key K, loader LoaderFunc[K, V]) (V, error)
func (c *ShardCache[K, V]) Stats() CacheStats
func (c *ShardCache[K, V]) Close() error
```

## 验收挑战点

测试用例会模拟真实的“脏环境”：
- 瞬间发起 10,000 个针对同一个 key 的并发 `GetOrLoad` 调用。
- 注入一个需要休眠 10 秒才会返回结果的 `LoaderFunc`，观察你的缓存锁是否会把其他完全不相关 key 的并发请求也阻塞住。
- 限制你的组件不能泄漏 goroutine 或内存。