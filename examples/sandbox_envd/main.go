package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox"
)

func main() {
	apiKey := os.Getenv("Qiniu_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 Qiniu_API_KEY 环境变量")
	}

	apiURL := os.Getenv("Qiniu_SANDBOX_API_URL")

	c, err := sandbox.NewClient(&sandbox.Config{
		APIKey:   apiKey,
		Endpoint: apiURL,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. 列出模板，选取第一个可用模板
	templates, err := c.ListTemplates(ctx, nil)
	if err != nil {
		log.Fatalf("列出模板失败: %v", err)
	}

	var templateID string
	for _, tmpl := range templates {
		if tmpl.BuildStatus == sandbox.BuildStatusReady || tmpl.BuildStatus == sandbox.BuildStatusUploaded {
			templateID = tmpl.TemplateID
			break
		}
	}
	if templateID == "" {
		log.Fatal("没有构建成功的模板")
	}
	fmt.Printf("使用模板: %s\n", templateID)

	// 2. 创建沙箱并等待就绪（附带 Metadata 和 NetworkConfig）
	timeout := int32(120)
	meta := sandbox.Metadata{"env": "dev", "team": "backend"}
	network := sandbox.NetworkConfig{
		AllowPublicTraffic: boolPtr(true),
	}
	sb, _, err := c.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: templateID,
		Timeout:    &timeout,
		Metadata:   &meta,
		Network:    &network,
	}, sandbox.WithPollInterval(2*time.Second))
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	fmt.Printf("沙箱已就绪: %s\n", sb.ID())

	// 验证 Metadata
	info, err := sb.GetInfo(ctx)
	if err != nil {
		log.Fatalf("获取沙箱信息失败: %v", err)
	}
	if info.Metadata != nil {
		fmt.Printf("Metadata: %v\n", *info.Metadata)
	}

	defer func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			log.Printf("终止沙箱失败: %v", err)
		} else {
			fmt.Printf("沙箱 %s 已终止\n", sb.ID())
		}
	}()

	// 3. 获取端口域名
	host := sb.GetHost(8080)
	fmt.Printf("端口 8080 访问地址: %s\n", host)

	// 4. 文件系统操作
	fmt.Println("\n--- 文件系统操作 ---")

	// 写入文件
	_, err = sb.Files().Write(ctx, "/tmp/hello.txt", []byte("Hello from Go SDK!\n"))
	if err != nil {
		log.Fatalf("写入文件失败: %v", err)
	}
	fmt.Println("文件已写入: /tmp/hello.txt")

	// 读取文件
	content, err := sb.Files().Read(ctx, "/tmp/hello.txt")
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}
	fmt.Printf("文件内容: %s", string(content))

	// 创建目录
	_, err = sb.Files().MakeDir(ctx, "/tmp/mydir")
	if err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}
	fmt.Println("目录已创建: /tmp/mydir")

	// 列出目录
	entries, err := sb.Files().List(ctx, "/tmp")
	if err != nil {
		log.Fatalf("列出目录失败: %v", err)
	}
	fmt.Printf("/tmp 目录内容 (%d 项):\n", len(entries))
	for _, e := range entries {
		fmt.Printf("  %s %s (%s, %d bytes)\n", e.Type, e.Name, e.Permissions, e.Size)
	}

	// 批量写入文件
	fmt.Println("\n--- 批量写入文件 ---")
	files := []sandbox.WriteEntry{
		{Path: "/tmp/batch-a.txt", Data: []byte("file A content")},
		{Path: "/tmp/batch-b.txt", Data: []byte("file B content")},
		{Path: "/tmp/batch-c.txt", Data: []byte("file C content")},
	}
	infos, err := sb.Files().WriteFiles(ctx, files)
	if err != nil {
		log.Fatalf("批量写入失败: %v", err)
	}
	for _, fi := range infos {
		fmt.Printf("已写入: %s (%d bytes)\n", fi.Path, fi.Size)
	}

	// ReadText — 读取文件为字符串
	fmt.Println("\n--- ReadText ---")
	text, err := sb.Files().ReadText(ctx, "/tmp/batch-a.txt")
	if err != nil {
		log.Fatalf("ReadText 失败: %v", err)
	}
	fmt.Printf("ReadText 结果: %q\n", text)

	// ReadStream — 流式读取文件
	fmt.Println("\n--- ReadStream ---")
	rc, err := sb.Files().ReadStream(ctx, "/tmp/batch-b.txt")
	if err != nil {
		log.Fatalf("ReadStream 失败: %v", err)
	}
	streamData, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		log.Fatalf("读取流失败: %v", err)
	}
	fmt.Printf("ReadStream 结果: %q\n", string(streamData))

	// Exists — 检查文件是否存在
	fmt.Println("\n--- Exists / GetInfo / Rename / Remove ---")
	exists, err := sb.Files().Exists(ctx, "/tmp/batch-a.txt")
	if err != nil {
		log.Fatalf("Exists 失败: %v", err)
	}
	fmt.Printf("Exists(/tmp/batch-a.txt) = %v\n", exists)

	// GetInfo — 获取文件元信息
	fileInfo, err := sb.Files().GetInfo(ctx, "/tmp/batch-a.txt")
	if err != nil {
		log.Fatalf("GetInfo 失败: %v", err)
	}
	fmt.Printf("GetInfo: name=%s, type=%s, size=%d, mode=%s\n",
		fileInfo.Name, fileInfo.Type, fileInfo.Size, fileInfo.Permissions)

	// Rename — 重命名文件
	renamedInfo, err := sb.Files().Rename(ctx, "/tmp/batch-c.txt", "/tmp/batch-c-renamed.txt")
	if err != nil {
		log.Fatalf("Rename 失败: %v", err)
	}
	fmt.Printf("Rename: %s -> %s\n", "/tmp/batch-c.txt", renamedInfo.Path)

	// Remove — 删除文件
	if err := sb.Files().Remove(ctx, "/tmp/batch-c-renamed.txt"); err != nil {
		log.Fatalf("Remove 失败: %v", err)
	}
	exists, err = sb.Files().Exists(ctx, "/tmp/batch-c-renamed.txt")
	if err != nil {
		log.Fatalf("Exists 失败: %v", err)
	}
	fmt.Printf("Remove 后 Exists(/tmp/batch-c-renamed.txt) = %v\n", exists)

	// WatchDir — 监听目录文件变更
	fmt.Println("\n--- WatchDir ---")

	// 创建监听目录
	_, err = sb.Files().MakeDir(ctx, "/tmp/watch-test")
	if err != nil {
		log.Fatalf("创建监听目录失败: %v", err)
	}

	// 启动目录监听
	wh, err := sb.Files().WatchDir(ctx, "/tmp/watch-test", sandbox.WithRecursive(true))
	if err != nil {
		log.Fatalf("WatchDir 失败: %v", err)
	}
	fmt.Println("已开始监听 /tmp/watch-test（递归）")

	// 在监听目录中触发文件变更
	_, err = sb.Files().Write(ctx, "/tmp/watch-test/watch-file.txt", []byte("watched content"))
	if err != nil {
		log.Fatalf("写入监听文件失败: %v", err)
	}
	fmt.Println("已写入: /tmp/watch-test/watch-file.txt")

	// 收集事件（等待最多 3 秒）
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	eventCount := 0
loop:
	for {
		select {
		case ev, ok := <-wh.Events():
			if !ok {
				break loop
			}
			eventCount++
			fmt.Printf("  事件: type=%s, name=%s\n", ev.Type, ev.Name)
		case <-timer.C:
			break loop
		}
	}
	fmt.Printf("共收到 %d 个事件\n", eventCount)

	// 停止监听
	wh.Stop()
	if err := wh.Err(); err != nil {
		fmt.Printf("监听错误: %v\n", err)
	} else {
		fmt.Println("监听已停止")
	}

	// 5. 执行命令
	fmt.Println("\n--- 命令执行 ---")

	result, err := sb.Commands().Run(ctx, "echo hello world")
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("命令: echo hello world\n")
	fmt.Printf("退出码: %d\n", result.ExitCode)
	fmt.Printf("stdout: %s", result.Stdout)

	// 带环境变量的命令
	result, err = sb.Commands().Run(ctx, "echo $MY_VAR",
		sandbox.WithEnvs(map[string]string{"MY_VAR": "sandbox-value"}),
	)
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("命令: echo $MY_VAR (MY_VAR=sandbox-value)\n")
	fmt.Printf("stdout: %s", result.Stdout)

	// WithCwd — 指定工作目录
	result, err = sb.Commands().Run(ctx, "pwd", sandbox.WithCwd("/tmp"))
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("命令: pwd (cwd=/tmp)\nstdout: %s", result.Stdout)

	// WithTimeout — 命令超时
	result, err = sb.Commands().Run(ctx, "echo fast", sandbox.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("命令: echo fast (timeout=5s)\nstdout: %s", result.Stdout)

	// WithOnStdout / WithOnStderr — 实时输出回调
	fmt.Println("\n--- 实时输出回调 ---")
	var stdoutChunks, stderrChunks int
	result, err = sb.Commands().Run(ctx, "echo out-line && echo err-line >&2",
		sandbox.WithOnStdout(func(data []byte) { stdoutChunks++ }),
		sandbox.WithOnStderr(func(data []byte) { stderrChunks++ }),
	)
	if err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
	fmt.Printf("stdout 回调次数: %d, stderr 回调次数: %d\n", stdoutChunks, stderrChunks)
	fmt.Printf("stdout: %sstderr: %s", result.Stdout, result.Stderr)

	// Start / List / Kill — 后台命令管理
	fmt.Println("\n--- 后台命令 (Start / List / Kill) ---")
	handle, err := sb.Commands().Start(ctx, "sleep 30", sandbox.WithTag("bg-sleep"))
	if err != nil {
		log.Fatalf("Start 失败: %v", err)
	}
	// 等待 PID 分配
	if _, err := handle.WaitPID(ctx); err != nil {
		log.Fatalf("WaitPID 失败: %v", err)
	}
	fmt.Printf("后台命令已启动: PID=%d\n", handle.PID())

	// List — 列出运行中的进程
	processes, err := sb.Commands().List(ctx)
	if err != nil {
		log.Fatalf("List 失败: %v", err)
	}
	fmt.Printf("运行中的进程 (%d 个):\n", len(processes))
	for _, p := range processes {
		tag := "<none>"
		if p.Tag != nil {
			tag = *p.Tag
		}
		fmt.Printf("  PID=%d, cmd=%s, tag=%s\n", p.PID, p.Cmd, tag)
	}

	// Kill — 终止后台进程
	if err := sb.Commands().Kill(ctx, handle.PID()); err != nil {
		log.Fatalf("Kill 失败: %v", err)
	}
	fmt.Printf("进程 PID=%d 已终止\n", handle.PID())

	// 6. 下载/上传 URL
	fmt.Println("\n--- 文件 URL ---")
	downloadURL := sb.DownloadURL("/tmp/hello.txt")
	fmt.Printf("下载 URL: %s\n", downloadURL)

	uploadURL := sb.UploadURL("/tmp/upload.txt")
	fmt.Printf("上传 URL: %s\n", uploadURL)

	// 通过 Files().Write() / Files().Read() 进行文件上传下载
	writeContent := []byte("uploaded via Files().Write()\n")
	if _, err := sb.Files().Write(ctx, "/tmp/upload-test.txt", writeContent); err != nil {
		log.Fatalf("Files().Write 失败: %v", err)
	}
	fmt.Println("Files().Write 成功: /tmp/upload-test.txt")

	readContent, err := sb.Files().Read(ctx, "/tmp/upload-test.txt")
	if err != nil {
		log.Fatalf("Files().Read 失败: %v", err)
	}
	fmt.Printf("Files().Read 结果: %q\n", string(readContent))

	// 7. PTY 终端
	fmt.Println("\n--- PTY 终端 ---")

	// Create — 创建 PTY 会话
	var ptyOutput []byte
	ptyHandle, err := sb.Pty().Create(ctx, sandbox.PtySize{Cols: 80, Rows: 24},
		sandbox.WithOnStdout(func(data []byte) {
			ptyOutput = append(ptyOutput, data...)
		}),
	)
	if err != nil {
		log.Fatalf("Pty.Create 失败: %v", err)
	}
	if _, err := ptyHandle.WaitPID(ctx); err != nil {
		log.Fatalf("WaitPID 失败: %v", err)
	}
	fmt.Printf("PTY 已创建: PID=%d\n", ptyHandle.PID())

	// SendInput — 向 PTY 发送输入
	if err := sb.Pty().SendInput(ctx, ptyHandle.PID(), []byte("echo pty-hello\n")); err != nil {
		log.Fatalf("Pty.SendInput 失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("PTY 输出片段: %q\n", truncate(string(ptyOutput), 200))

	// Resize — 调整终端大小
	if err := sb.Pty().Resize(ctx, ptyHandle.PID(), sandbox.PtySize{Cols: 120, Rows: 40}); err != nil {
		log.Fatalf("Pty.Resize 失败: %v", err)
	}
	fmt.Println("PTY 已调整为 120x40")

	// Kill — 终止 PTY 会话
	if err := sb.Pty().Kill(ctx, ptyHandle.PID()); err != nil {
		log.Fatalf("Pty.Kill 失败: %v", err)
	}
	fmt.Printf("PTY PID=%d 已终止\n", ptyHandle.PID())
}

// boolPtr 返回 bool 值的指针。
func boolPtr(v bool) *bool { return &v }

// truncate 截断字符串到指定最大长度。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
