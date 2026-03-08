# M3 组件设计题：高抽象高复用的并发缓存组件

## 场景背景

你需要实现一个可复用的通用组件，供多个业务服务共享：

- 网关层：按 key 缓存鉴权结果
- 推荐层：缓存特征计算结果
- 风控层：缓存规则命中结果

该组件必须既能在高并发下稳定运行，又具备良好的抽象与封装，避免每个业务重复造轮子。

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

type LoaderFunc[K comparable, V any] func(ctx context.Context, key K) (value V, ttl time.Duration, err error)

type ShardCache[K comparable, V any] struct { /* unexported fields */ }

func NewShardCache[K comparable, V any](cfg CacheConfig) (*ShardCache[K, V], error)
func (c *ShardCache[K, V]) Get(ctx context.Context, key K) (V, bool)
func (c *ShardCache[K, V]) Set(ctx context.Context, key K, value V, ttl time.Duration)
func (c *ShardCache[K, V]) Delete(ctx context.Context, key K)
func (c *ShardCache[K, V]) GetOrLoad(ctx context.Context, key K, loader LoaderFunc[K, V]) (V, error)
func (c *ShardCache[K, V]) Stats() CacheStats
func (c *ShardCache[K, V]) Close() error
```

## 功能与设计要求

1. **高抽象复用**  
   - 必须使用泛型 `K comparable, V any`。
   - 配置通过 `CacheConfig` 注入，禁止硬编码阈值/时长/容量。

2. **并发性能**  
   - 必须采用分片（Sharding）减少锁竞争。
   - `GetOrLoad` 必须做到 **same-key singleflight**（同 key 并发只触发一次 loader）。
   - 不允许在持锁区执行慢 loader 调用。

3. **封装与可靠性**  
   - 结构体字段保持非导出，避免外部破坏状态。
   - `Get` / `GetOrLoad` 必须支持 TTL 过期语义。
   - 当单分片超过 `MaxEntriesPerShard` 时，必须执行驱逐（至少可观测为发生了 evictions）。

4. **可量化评估目标**  
   - 通过 race 检测；
   - 通过并发 contract tests；
   - benchmark 有可读结果（用于评分参考）。

## 失败判定（关键）

- 任一 contract test 失败；
- 出现 Data Race；
- 未实现 same-key singleflight；
- 在 `GetOrLoad` 中持锁执行慢 loader，导致并发阻塞明显。
