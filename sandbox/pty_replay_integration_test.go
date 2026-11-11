//go:build integration

package sandbox

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestIntegrationPtyReplayLifecycle 使用明确指定的新版 envd 模板验收真实重连。
func TestIntegrationPtyReplayLifecycle(t *testing.T) {
	templateID := os.Getenv("QINIU_SANDBOX_PTY_REPLAY_TEMPLATE")
	if templateID == "" {
		t.Skip("需要设置包含新版 envd 的 QINIU_SANDBOX_PTY_REPLAY_TEMPLATE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	client := testClient(t)
	ttl := int32(120)
	sb, _, err := client.CreateAndWait(ctx, CreateParams{TemplateID: templateID, Timeout: &ttl})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		if err := sb.Kill(cleanup); err != nil {
			t.Errorf("清理 Sandbox 失败: %v", err)
		}
	})
	user := os.Getenv("QINIU_SANDBOX_PTY_REPLAY_USER")
	if user == "" {
		user = DefaultUser
	}
	pty := sb.Pty()
	tag := fmt.Sprintf("sdk-replay-%d", time.Now().UnixNano())
	stream, err := pty.CreateReplay(ctx, PtySize{Cols: 80, Rows: 24}, WithCommandUser(user), WithTag(tag))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	ref := stream.Reference
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := pty.KillReplay(cleanup, ref, WithCommandUser(user)); err != nil {
			t.Errorf("清理 PTY 失败: %v", err)
		}
	})
	readThrough := func(s *PtyReplayStream, marker string) (string, uint64) {
		t.Helper()
		var output strings.Builder
		for {
			event, err := s.Receive()
			if err != nil {
				t.Fatal(err)
			}
			if event.End != nil {
				t.Fatalf("PTY 意外退出: %+v", event.End)
			}
			output.Write(event.Data)
			if output.Len() > 64*1024 {
				t.Fatal("未在有界输出中找到标记")
			}
			if strings.Contains(output.String(), marker) {
				return output.String(), event.Offset
			}
		}
	}
	writeMarker := func(marker string) {
		t.Helper()
		// 命令回显不包含拼接后的标记，只有实际 printf 执行结果才匹配。
		mid := len(marker) / 2
		command := fmt.Sprintf("printf '%%s%%s\\n' '%s' '%s'\n", marker[:mid], marker[mid:])
		if err := pty.SendReplayInput(ctx, ref, []byte(command), WithCommandUser(user)); err != nil {
			t.Fatal(err)
		}
	}
	before := tag + "-before"
	writeMarker(before)
	_, cursor := readThrough(stream, before)
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pty.ResizeReplay(ctx, ref, PtySize{Cols: 100, Rows: 30}, WithCommandUser(user)); err != nil {
		t.Fatal(err)
	}
	after := tag + "-after-disconnect"
	writeMarker(after)
	refs, err := pty.FindReplay(ctx, tag, WithCommandUser(user))
	if err != nil || len(refs) != 1 || refs[0] != ref {
		t.Fatalf("创建身份未保留: %v, %v", refs, err)
	}
	resumed, err := pty.ConnectReplay(ctx, ref, &cursor, WithCommandUser(user))
	if err != nil {
		t.Fatal(err)
	}
	defer resumed.Close()
	if resumed.Reference != ref || resumed.Offset != cursor {
		t.Fatalf("重连身份或游标变化: %+v", resumed)
	}
	output, end := readThrough(resumed, after)
	if strings.Contains(output, before) || end <= cursor {
		t.Fatalf("重复输出或游标未前进: start=%d end=%d output=%q", cursor, end, output)
	}
}
