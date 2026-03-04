检查发布版本号是否在三处保持一致。

执行步骤：

1. 从 `conf/conf.go` 读取 `const Version = "x.y.z"`
2. 从 `CHANGELOG.md` 检查是否包含 `## x.y.z` 条目
3. 从 `README.md` 检查是否包含 `require github.com/qiniu/go-sdk/v7 vx.y.z`

输出版本号和三处的匹配状态表格：

| 位置 | 期望 | 状态 |
|------|------|------|
| conf/conf.go | vX.Y.Z | 匹配/不匹配 |
| CHANGELOG.md | ## X.Y.Z | 匹配/不匹配 |
| README.md | require ... vX.Y.Z | 匹配/不匹配 |

如果有不一致，给出具体的修复建议。
