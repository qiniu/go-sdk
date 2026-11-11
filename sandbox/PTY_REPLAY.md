# 可重连的 PTY

`Client.Attach` 只读取现有 Sandbox 的地址和 envd 凭证，不恢复暂停实例，也不延长生命周期。
`Pty.CreateReplay` 在创建前查询 envd 的 `pty_replay_supported` 能力；旧 envd 返回
`ErrPtyReplayUnsupported`，不会创建不支持重放的进程。

```go
sb, err := client.Attach(ctx, sandboxID)
if err != nil { return err }
stream, err := sb.Pty().CreateReplay(ctx, sandbox.PtySize{Cols: 100, Rows: 28}, sandbox.WithTag(uniqueTag))
if err != nil { return err }
reference, baseline := stream.Reference, stream.Offset
// 将 reference 和 baseline 持久化；关闭流只断开订阅。
_ = stream.Close()
resumed, err := sb.Pty().ConnectReplay(ctx, reference, &baseline)
if err != nil { return err }
defer resumed.Close()
for {
    event, err := resumed.Receive()
    if err != nil { return err }
    if event.End != nil { break }
    // event.Offset 是该段输出之后的绝对字节偏移。
    // 只在下游接受输出后保存游标；Heartbeat 同样携带当前偏移。
}
```

`SendReplayInput`、`ResizeReplay`、`KillReplay` 都同时校验 PID 和 StreamID，避免误操作
重用 PID 的其他进程。输入失败可能意味着执行结果未知，调用方不得自动重发输入。
`FindReplay` 使用创建前保存的唯一标签查询进程，用于处理未知创建结果；查不到标签不能证明
一个仍在处理的创建请求已经失败。

envd 保留每个 PTY 最近 64 KiB 的输出和退出后 30 秒的尾部/结果。超出保留窗口会返回
Connect `out_of_range`，调用方必须明确结束会话，不能把缺口当作正常重连。调用方负责持久化
会话权限、执行对象绑定和清理意图。RPC context 仅约束当前订阅；配套 envd 不会因创建
RPC 超时而终止 replay PTY，进程由显式 Kill 或 Sandbox 生命周期回收。

新方法使用 CommandOption 中的环境变量、工作目录、用户和标签；输出由 `Receive` 拉取。
原有 `Pty.Create`、`Pty.Connect`、Commands API 的签名和行为保持兼容。

协议源位于 envd 的 `spec/process/process.proto`，并同步到 api-specs 提交 `58ee2f835d135e62ebc32bac3e5aa91cbfc21df6`。
本改动的 process.pb.go 从该协议通过 buf 生成；提交固定对应的 api-specs 版本，以保证可复现生成。

能力查询、重连、标签查询和三种控制方法均接受可选的 `WithCommandUser`；自定义模板
使用 `root` 等账户时，每次调用传入相同选项，创建前的能力查询也使用这个账户。
这些后续方法仅使用 CommandOption 的用户字段。输入、resize 和 kill 每次先查询能力，
旧 envd 或能力查询失败时不发送修改请求。该检查不保存在节点缓存中。
初始输出已超出回放窗口时，创建仍先返回带 `Reference` 的流；后续 `Receive` 返回缺口错误，
调用方可使用已取得的身份清理 PTY，游标不会自动前移。

真实生命周期验收使用 `TestIntegrationPtyReplayLifecycle`，额外设置
`QINIU_SANDBOX_PTY_REPLAY_TEMPLATE` 为已经部署新版 envd 的模板，可选
`QINIU_SANDBOX_PTY_REPLAY_USER` 指定 OS 用户。测试创建短期 Sandbox，验证创建、
断开订阅、无订阅时输入、保存游标重连及清理；未配置模板时跳过。
