//go:build integration

package sandbox

import (
	"bytes"
	"context"
	"image/png"
	"net/http"
	"net/url"
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
	if err := desktop.Click(ctx, MouseButtonLeft, &point); err != nil {
		t.Fatalf("Click 失败: %v", err)
	}
	if err := desktop.DoubleClick(ctx, &point); err != nil {
		t.Fatalf("DoubleClick 失败: %v", err)
	}
	if err := desktop.MouseDown(ctx, MouseButtonRight); err != nil {
		t.Fatalf("MouseDown 失败: %v", err)
	}
	if err := desktop.MouseUp(ctx, MouseButtonRight); err != nil {
		t.Fatalf("MouseUp 失败: %v", err)
	}
	if err := desktop.Drag(ctx, point, Point{X: 120, Y: 120}); err != nil {
		t.Fatalf("Drag 失败: %v", err)
	}
	if err := desktop.Scroll(ctx, ScrollDirectionDown, 1); err != nil {
		t.Fatalf("Scroll 失败: %v", err)
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
	activeWindowID, err := desktop.ActiveWindowID(ctx)
	if err != nil {
		t.Fatalf("ActiveWindowID 失败: %v", err)
	}
	if activeWindowID == "" {
		t.Fatal("ActiveWindowID 返回空值")
	}
	if err := desktop.TypeText(ctx, "echo desktop-integration", nil); err != nil {
		t.Fatalf("TypeText 失败: %v", err)
	}
	if err := desktop.Press(ctx, "enter"); err != nil {
		t.Fatalf("Press 失败: %v", err)
	}
	if err := desktop.Wait(ctx, 100*time.Millisecond); err != nil {
		t.Fatalf("Wait 失败: %v", err)
	}
	if err := desktop.Open(ctx, "https://example.com"); err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	if err := desktop.Wait(ctx, 500*time.Millisecond); err != nil {
		t.Fatalf("等待浏览器启动失败: %v", err)
	}
	if _, err := desktop.Screenshot(ctx); err != nil {
		t.Fatalf("操作后截图失败: %v", err)
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
	resp.Body.Close()

	parsedURL, err := url.Parse(streamURL)
	if err != nil {
		t.Fatalf("解析 noVNC URL 失败: %v", err)
	}
	if parsedURL.Path != "/vnc.html" || parsedURL.Query().Get("password") != authKey {
		t.Fatalf("noVNC URL 参数异常: %s", streamURL)
	}
	if err := desktop.Stream().Stop(ctx); err != nil {
		t.Fatalf("停止桌面流失败: %v", err)
	}
	if _, err := desktop.Stream().URL(nil); err == nil {
		t.Fatal("停止桌面流后 URL 仍可用")
	}

	noAuth := false
	if err := desktop.Stream().Start(ctx, &DesktopStreamOptions{
		VNCPort:     5901,
		WebPort:     6081,
		RequireAuth: &noAuth,
		WindowID:    activeWindowID,
	}); err != nil {
		t.Fatalf("使用自定义端口和窗口启动桌面流失败: %v", err)
	}
	if _, err := desktop.Stream().AuthKey(); err == nil {
		t.Fatal("禁用认证后 AuthKey 未返回错误")
	}
	noAuthURL, err := desktop.Stream().URL(&DesktopStreamURLOptions{
		AutoConnect: boolPointer(false),
		ViewOnly:    true,
		Resize:      DesktopStreamResizeRemote,
	})
	if err != nil {
		t.Fatalf("生成无认证 noVNC URL 失败: %v", err)
	}
	noAuthParsed, err := url.Parse(noAuthURL)
	if err != nil {
		t.Fatalf("解析无认证 noVNC URL 失败: %v", err)
	}
	if noAuthParsed.Query().Get("autoconnect") != "" || noAuthParsed.Query().Get("password") != "" ||
		noAuthParsed.Query().Get("view_only") != "true" || noAuthParsed.Query().Get("resize") != string(DesktopStreamResizeRemote) {
		t.Fatalf("无认证 noVNC URL 参数异常: %s", noAuthURL)
	}
	noAuthReq, err := http.NewRequestWithContext(ctx, http.MethodGet, noAuthURL, nil)
	if err != nil {
		t.Fatalf("创建无认证 noVNC 请求失败: %v", err)
	}
	noAuthResp, err := http.DefaultClient.Do(noAuthReq)
	if err != nil {
		t.Fatalf("请求无认证 noVNC 页面失败: %v", err)
	}
	noAuthResp.Body.Close()
	if noAuthResp.StatusCode != http.StatusOK {
		t.Fatalf("无认证 noVNC 页面状态码 = %d, want 200", noAuthResp.StatusCode)
	}
	if err := desktop.Stream().Stop(ctx); err != nil {
		t.Fatalf("停止自定义桌面流失败: %v", err)
	}
}

func boolPointer(value bool) *bool {
	return &value
}
