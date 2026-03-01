// Package sandbox 提供七牛云沙箱服务的 Go SDK，用于管理安全隔离的云端沙箱环境。
//
// 沙箱服务是一款专为 AI Agent 场景设计的运行时基础设施，提供安全隔离的云端环境来执行
// AI 生成的代码。通过系统级隔离机制，确保代码执行不会对宿主系统造成非法访问或篡改。
// 沙箱启动时间低于 200 毫秒，默认存活 5 分钟（最长 1 小时），支持暂停/恢复以持久化
// 文件系统和内存状态。
//
// 更多产品信息请参阅: https://developer.qiniu.com/las/13281/sandbox-overview
//
// # 核心概念
//
//   - Sandbox: 隔离的云端执行环境（轻量级虚拟化），支持 running、paused、killed 三种状态
//   - Template: 预构建的沙箱环境定义，包含基础镜像、依赖、文件和启动命令，实现亚秒级启动
//   - envd: 运行在沙箱内部的 agent 守护进程，通过 ConnectRPC 提供进程管理、文件系统操作和 PTY 终端服务
//
// # 快速开始
//
// 创建客户端并启动沙箱:
//
//	c, err := sandbox.NewClient(&sandbox.Config{
//	    APIKey: os.Getenv("Qiniu_API_KEY"),
//	})
//
//	timeout := int32(120)
//	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
//	    TemplateID: "base",
//	    Timeout:    &timeout,
//	}, sandbox.WithPollInterval(2*time.Second))
//
//	defer sb.Kill(ctx)
//
// # 沙箱生命周期
//
// Client 提供沙箱的创建、连接和列表操作:
//
//   - [Client.Create] / [Client.CreateAndWait]: 创建沙箱（后者会轮询等待就绪）
//   - [Client.Connect]: 连接到已有沙箱，可恢复已暂停的沙箱
//   - [Client.List] / [Client.ListV2]: 列出沙箱，支持按状态和元数据过滤
//
// Sandbox 实例提供生命周期管理:
//
//   - [Sandbox.Kill]: 终止沙箱
//   - [Sandbox.Pause]: 暂停沙箱（保留文件系统和内存状态）
//   - [Sandbox.SetTimeout]: 更新超时时间
//   - [Sandbox.Refresh]: 延长存活时间
//   - [Sandbox.GetInfo] / [Sandbox.IsRunning]: 查询沙箱状态
//   - [Sandbox.GetMetrics]: 获取 CPU、内存、磁盘等资源指标
//   - [Sandbox.GetLogs]: 获取沙箱日志
//   - [Sandbox.WaitForReady]: 轮询等待沙箱进入 running 状态
//
// # 命令执行
//
// 通过 [Sandbox.Commands] 在沙箱内执行终端命令:
//
//	// 同步执行
//	result, err := sb.Commands().Run(ctx, "echo hello",
//	    sandbox.WithEnvs(map[string]string{"MY_VAR": "value"}),
//	    sandbox.WithCwd("/tmp"),
//	    sandbox.WithTimeout(5*time.Second),
//	)
//	fmt.Println(result.Stdout)
//
//	// 异步执行（后台命令）
//	handle, err := sb.Commands().Start(ctx, "sleep 30", sandbox.WithTag("bg"))
//	handle.WaitPID(ctx)
//	sb.Commands().Kill(ctx, handle.PID())
//
// Commands 支持实时输出回调（[WithOnStdout] / [WithOnStderr]）、后台命令管理
// （[Commands.Start] / [Commands.List] / [Commands.Kill]）以及标准输入发送
// （[Commands.SendStdin]）。
//
// # 文件系统操作
//
// 通过 [Sandbox.Files] 进行文件读写:
//
//	// 写入和读取文件
//	sb.Files().Write(ctx, "/tmp/hello.txt", []byte("Hello!"))
//	content, err := sb.Files().Read(ctx, "/tmp/hello.txt")
//
//	// 批量写入
//	sb.Files().WriteFiles(ctx, []sandbox.WriteEntry{
//	    {Path: "/tmp/a.txt", Data: []byte("content A")},
//	    {Path: "/tmp/b.txt", Data: []byte("content B")},
//	})
//
//	// 目录操作
//	sb.Files().MakeDir(ctx, "/tmp/mydir")
//	entries, err := sb.Files().List(ctx, "/tmp")
//
//	// 监听目录变更
//	wh, err := sb.Files().WatchDir(ctx, "/tmp/watch", sandbox.WithRecursive(true))
//	for ev := range wh.Events() {
//	    fmt.Printf("event: %s %s\n", ev.Type, ev.Name)
//	}
//
// Filesystem 还提供 [Filesystem.ReadText]、[Filesystem.ReadStream]、
// [Filesystem.Exists]、[Filesystem.GetInfo]、[Filesystem.Rename]、
// [Filesystem.Remove] 等操作。
//
// # PTY 终端
//
// 通过 [Sandbox.Pty] 创建和管理伪终端会话:
//
//	ptyHandle, err := sb.Pty().Create(ctx, sandbox.PtySize{Cols: 80, Rows: 24},
//	    sandbox.WithOnStdout(func(data []byte) { fmt.Print(string(data)) }),
//	)
//	sb.Pty().SendInput(ctx, ptyHandle.PID(), []byte("ls -la\n"))
//	sb.Pty().Resize(ctx, ptyHandle.PID(), sandbox.PtySize{Cols: 120, Rows: 40})
//	sb.Pty().Kill(ctx, ptyHandle.PID())
//
// # 模板管理
//
// Client 提供模板的完整生命周期管理:
//
//   - [Client.ListTemplates] / [Client.GetTemplate]: 列出和查询模板
//   - [Client.CreateTemplate]: 创建模板（返回 templateID 和 buildID）
//   - [Client.UpdateTemplate] / [Client.DeleteTemplate]: 更新和删除模板
//   - [Client.StartTemplateBuild] / [Client.WaitForBuild]: 启动构建并等待完成
//   - [Client.GetTemplateBuildStatus] / [Client.GetTemplateBuildLogs]: 查询构建状态和日志
//   - [Client.ManageTemplateTags] / [Client.DeleteTemplateTags]: 管理模板标签
//   - [Client.GetTemplateByAlias]: 通过别名查找模板
//
// # 网络访问
//
// 沙箱默认允许访问互联网，可通过 [CreateParams] 的 Network 字段配置出站流量规则。
// 使用 [Sandbox.GetHost] 获取外部访问沙箱指定端口的域名。
//
// # 轮询选项
//
// [Client.CreateAndWait]、[Sandbox.WaitForReady] 和 [Client.WaitForBuild] 支持
// 通过 [PollOption] 自定义轮询行为:
//
//   - [WithPollInterval]: 设置轮询间隔
//   - [WithBackoff]: 启用指数退避
//   - [WithOnPoll]: 注册轮询回调（用于日志或进度展示）
package sandbox
