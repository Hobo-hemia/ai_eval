# M3 测试床

本目录用于验证 `m3_component` 产出的 `m3_result.go` 是否满足高抽象、高复用与高并发性能要求。

## 覆盖目标

- 泛型契约与配置校验（`CacheConfig`/`ShardCache[K,V]`）
- TTL 与驱逐行为可验证
- same-key singleflight（并发去重）
- slow loader 不阻塞其他 key（锁粒度合理）
- race 检测无数据竞争
- benchmark 输出（用于评分参考）

## 目录说明

- `harness/`: 合同测试与 benchmark
- `run_full_chain.sh`: 一键构建/测试/benchmark，日志写入 `eval_records/<model_dir>/m3_component/`

## 执行方式

在仓库根目录执行：

```bash
bash modules/m3_component/tests/run_full_chain.sh <model_dir>
```

会产出：

- `eval_records/<model_dir>/m3_component/m3_build.log`
- `eval_records/<model_dir>/m3_component/m3_test.log`
