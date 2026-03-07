Role: 资深 Go 后端架构师
Task: 架构与脚手架代码生成

请阅读协议文件 @api.proto，并严格遵循当前工作区 .cursorrules 中的编码规范，生成对应的 Go 服务端骨架与基础业务逻辑。

【核心验收要求】：
1. 架构分层：请清晰定义 Handler 层及底层的 Service 接口定义，不要将所有逻辑揉杂在一个函数中。
2. 数据校验：必须使用 protoc-gen-validate 的逻辑，在 Handler 入口处调用生成的 Validate() 方法完成基础校验。
3. 规范要求：强制使用 gRPC 标准的 status.Errorf 进行错误包装和返回；命名严格遵循 Go 语言规范。

【输出要求】：
请直接输出完整的 Go 代码实现（包含 package 声明与 import）。无需任何开场白、结语或原理解释，确保可一键保存为单一 .go 文件并编译。
