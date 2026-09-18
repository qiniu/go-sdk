//go:build integration

package sandbox

import (
	"bytes"
	"context"
	"image/png"
	"net/http"
	"testing"
	"time"
)

func TestIntegrationDesktop(t *testing.T) {
	client := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	timeout := int32(180)
	allowPublicTraffic := true
	desktop, err := client.CreateDesktop(ctx, DesktopCreateParams{
		CreateParams: CreateParams{
			Timeout: &timeout,
			Network: &NetworkConfig{AllowPublicTraffic: &allowPublicTraffic},
		},
		Resolution: ScreenSize{Width: 1024, Height: 768},
	}, WithPollInterval(2*time.Second))
	if err != nil {
		t.Fatalf("CreateDesktop 失败: %v", err)
	}
	t.Cleanup(func() {
		streamCtx, streamCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := desktop.Stream().Stop(streamCtx); err != nil {
			t.Logf("停止桌面流失败: %v", err)
		}
		streamCancel()
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := desktop.Kill(killCtx); err != nil {
			t.Logf("清理 Desktop Sandbox %s 失败: %v", desktop.ID(), err)
		}
	})

	size, err := desktop.ScreenSize(ctx)
	if err != nil {
		t.Fatalf("ScreenSize 失败: %v", err)
	}
	if size != (ScreenSize{Width: 1024, Height: 768}) {
		t.Fatalf("ScreenSize = %+v, want 1024x768", size)
	}

	screenshot, err := desktop.Screenshot(ctx)
	if err != nil {
		t.Fatalf("Screenshot 失败: %v", err)
	}
	imageConfig, err := png.DecodeConfig(bytes.NewReader(screenshot))
	if err != nil {
		t.Fatalf("截图不是有效 PNG: %v", err)
	}
	if imageConfig.Width != 1024 || imageConfig.Height != 768 {
		t.Fatalf("截图尺寸 = %dx%d, want 1024x768", imageConfig.Width, imageConfig.Height)
	}

	point := Point{X: 100, Y: 100}
	if err := desktop.MoveMouse(ctx, point); err != nil {
		t.Fatalf("MoveMouse 失败: %v", err)
	}
	actualPoint, err := desktop.CursorPosition(ctx)
	if err != nil {
		t.Fatalf("CursorPosition 失败: %v", err)
	}
	if actualPoint != point {
		t.Fatalf("CursorPosition = %+v, want %+v", actualPoint, point)
	}

	if err := desktop.Launch(ctx, "xfce4-terminal"); err != nil {
		t.Fatalf("Launch 失败: %v", err)
	}
	var windowIDs []string
	err = desktop.waitUntil(ctx, 15*time.Second, func(ctx context.Context) (bool, error) {
		var queryErr error
		windowIDs, queryErr = desktop.ApplicationWindows(ctx, "xfce4-terminal")
		return len(windowIDs) > 0, queryErr
	})
	if err != nil {
		t.Fatalf("等待终端窗口失败: %v", err)
	}
	if _, err := desktop.WindowTitle(ctx, windowIDs[0]); err != nil {
		t.Fatalf("WindowTitle 失败: %v", err)
	}
	if err := desktop.TypeText(ctx, "echo desktop-integration", nil); err != nil {
		t.Fatalf("TypeText 失败: %v", err)
	}
	if err := desktop.Press(ctx, "enter"); err != nil {
		t.Fatalf("Press 失败: %v", err)
	}

	if err := desktop.Stream().Start(ctx, nil); err != nil {
		t.Fatalf("启动桌面流失败: %v", err)
	}
	authKey, err := desktop.Stream().AuthKey()
	if err != nil {
		t.Fatalf("AuthKey 失败: %v", err)
	}
	streamURL, err := desktop.Stream().URL(&DesktopStreamURLOptions{AuthKey: authKey})
	if err != nil {
		t.Fatalf("URL 失败: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		t.Fatalf("创建 noVNC 请求失败: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("请求 noVNC 页面失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("noVNC 页面状态码 = %d, want 200", resp.StatusCode)
	}
}
