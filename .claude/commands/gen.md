运行代码生成并检查变更。

执行步骤：

1. 先运行 `git diff --stat` 记录当前未提交的变更状态
2. 运行 `make generate` 执行 API 代码生成（storagev2、iam、media、audit）
3. 运行 `git diff --stat` 对比生成前后的变更
4. 如果有新的变更，列出生成产生的文件差异
5. 运行 `go build ./...` 确认编译通过

注意：如果用户提到 sandbox，则运行 `make generate-sandbox` 而非 `make generate`。
