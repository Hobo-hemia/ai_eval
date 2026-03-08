# eval_records 说明

`eval_records` 由 `ai_eval_init` 按模型与模块创建目录，`ai_eval` 运行后写入最终评测产物。

按 `模型目录 / 模块` 维度隔离存储：

- `mX_result.go`: 被测模型生成代码
- `mX_build.log`: 编译日志
- `mX_test.log`: 测试日志
- `score.json`: 裁判评分结果

说明：

- 模型目录名是安全化后的字符串（由 `ModelDirName` 生成）。
- 初始化阶段只建目录，不预创建工作文件。
