Role: 资深 Go 基础组件架构工程师
Task: 设计高抽象、高复用、高并发缓存组件

请阅读并严格遵循：
- @input/rate_limit_spec.md
- @.cursorrules

你需要输出一个可复用组件 `ShardCache[K, V]`，用于多业务共享。重点不是业务逻辑本身，而是：
1) 抽象能力（泛型 + 配置化）
2) 性能能力（分片 + singleflight + 锁粒度）
3) 工程封装（对外 API 稳定、内部状态可控、可观测统计）

【核心验收要求】：
1. 必须支持泛型 `K comparable, V any`，不得退化为写死 string/int 版本。
2. `GetOrLoad` 必须保证 same-key singleflight，并且 slow loader 不可在持锁区执行。
3. 必须实现 TTL、驱逐、并发安全统计（Hits/Misses/LoadSuccess/LoadFailures/Evictions）。
4. 必须在关键修复/设计处加注释：`// BUGFIX: [根因与修复逻辑]`（至少两处）。

【输出要求】：
仅输出两个代码块，禁止解释文字：
1) 第一段：`package result` 的组件实现代码（供 m3_result.go 落盘）
2) 第二段：`package result` 的单测代码（table-driven + 至少一个 benchmark）
