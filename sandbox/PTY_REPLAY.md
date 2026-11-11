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
