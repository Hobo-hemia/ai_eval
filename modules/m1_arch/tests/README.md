# M1 测试床

本目录用于验证 M1 协议改造结果（`m1_result.proto`）是否符合：

- 从产品抽象需求提炼出后端协议主链路
- 对 v1 协议做有效演进（重命名、增删字段）
- 对 5 个功能点做接口收敛（新增 rpc < 5）
- 协议结构清晰，可扩展

## 执行方式

```bash
bash modules/m1_arch/tests/run_full_chain.sh <model_dir>
```

## 产物

- `eval_records/<model_dir>/m1_arch/m1_build.log`
- `eval_records/<model_dir>/m1_arch/m1_test.log`
