package sandbox

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultDesktopVNCPort       = 5900
	defaultDesktopWebPort       = 6080
	defaultDesktopStreamTimeout = 10 * time.Second
	desktopStreamKeyLength      = 16
)

const desktopStreamKeyAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// DesktopStreamResizeMode 表示 noVNC 客户端的画面缩放方式。
type DesktopStreamResizeMode string

const (
	// DesktopStreamResizeOff 禁用画面缩放。
	DesktopStreamResizeOff DesktopStreamResizeMode = "off"
	// DesktopStreamResizeScale 按浏览器窗口缩放画面。
	DesktopStreamResizeScale DesktopStreamResizeMode = "scale"
	// DesktopStreamResizeRemote 请求远端桌面调整分辨率。
	DesktopStreamResizeRemote DesktopStreamResizeMode = "remote"
)

// DesktopStreamOptions 是 noVNC 桌面流启动选项。
type DesktopStreamOptions struct {
	// VNCPort 是 x11vnc 监听端口，默认 5900，取值范围为 1-65535，
	// 且必须与 WebPort 不同。
	VNCPort int
	// WebPort 是 noVNC Web 服务监听端口，默认 6080，取值范围为 1-65535，
	// 且必须与 VNCPort 不同。
	WebPort int
	// RequireAuth 控制 VNC 密码认证，nil 表示启用认证。
	RequireAuth *bool
	// WindowID 非空时仅共享指定的 X11 窗口。
	WindowID string
}

// DesktopStreamURLOptions 是 noVNC 页面 URL 选项。
type DesktopStreamURLOptions struct {
	// AutoConnect 控制页面打开后是否自动连接，nil 表示启用。
	AutoConnect *bool
	// ViewOnly 禁止通过 noVNC 页面操作桌面。
	ViewOnly bool
	// Resize 指定画面缩放方式，空值表示 scale。
	Resize DesktopStreamResizeMode
	// AuthKey 是显式写入 URL 的 VNC 密码。查询参数可能被浏览器历史、
	// 代理和日志记录；仅应在调用方接受该风险时设置。
	AuthKey string
}

// DesktopStream 管理 Desktop 的 x11vnc 和 noVNC 进程。
// 当 Sandbox 使用私有流量访问时，平台要求通过 Header 携带流量令牌，
// 浏览器直接打开 noVNC URL 无法附加该 Header，应由调用方提供鉴权代理。
type DesktopStream struct {
	desktop *Desktop

	mu        sync.Mutex
	running   bool
	vncPort   int
	webPort   int
	password  string
	noVNCPID  uint32
	requirePW bool
}

func newDesktopStream(desktop *Desktop) *DesktopStream {
	return &DesktopStream{desktop: desktop}
}

func normalizeDesktopStreamOptions(options *DesktopStreamOptions) (DesktopStreamOptions, bool, error) {
	normalized := DesktopStreamOptions{
		VNCPort: defaultDesktopVNCPort,
		WebPort: defaultDesktopWebPort,
	}
	requireAuth := true
	if options != nil {
		if options.VNCPort != 0 {
			normalized.VNCPort = options.VNCPort
		}
		if options.WebPort != 0 {
			normalized.WebPort = options.WebPort
		}
		normalized.RequireAuth = options.RequireAuth
		normalized.WindowID = options.WindowID
		if options.RequireAuth != nil {
			requireAuth = *options.RequireAuth
		}
	}
	if normalized.VNCPort < 1 || normalized.VNCPort > 65535 {
		return DesktopStreamOptions{}, false, fmt.Errorf("desktop VNC port must be between 1 and 65535")
	}
	if normalized.WebPort < 1 || normalized.WebPort > 65535 {
		return DesktopStreamOptions{}, false, fmt.Errorf("desktop web port must be between 1 and 65535")
	}
	if normalized.VNCPort == normalized.WebPort {
		return DesktopStreamOptions{}, false, fmt.Errorf("desktop VNC and web ports must differ")
	}
	return normalized, requireAuth, nil
}

func generateDesktopStreamKey(length int) (string, error) {
	key := make([]byte, length)
	upperBound := big.NewInt(int64(len(desktopStreamKeyAlphabet)))
	for i := range key {
		index, err := rand.Int(rand.Reader, upperBound)
		if err != nil {
			return "", fmt.Errorf("generate desktop stream auth key: %w", err)
		}
		key[i] = desktopStreamKeyAlphabet[index.Int64()]
	}
	return string(key), nil
}

func x11VNCProcessMatch(port int) string {
	return "[x]11vnc.*-rfbport " + strconv.Itoa(port)
}

func (s *DesktopStream) checkRunning(ctx context.Context, port int) (bool, error) {
	command := "pgrep -f " + shellEscape(x11VNCProcessMatch(port))
	result, err := s.desktop.commands.Run(ctx, command, s.desktop.commandOptions()...)
	if err != nil {
		return false, fmt.Errorf("desktop check stream: %w", err)
	}
	if result.ExitCode == 1 {
		return false, nil
	}
	if result.ExitCode != 0 {
		return false, &DesktopCommandError{Operation: "check stream", Result: result}
	}
	return strings.TrimSpace(result.Stdout) != "", nil
}

func (s *DesktopStream) cleanupCommand(ctx context.Context, operation, command string) error {
	result, err := s.desktop.commands.Run(ctx, command, s.desktop.commandOptions()...)
	if err != nil {
		return fmt.Errorf("desktop %s: %w", operation, err)
	}
	if result.ExitCode != 0 && result.ExitCode != 1 {
		return &DesktopCommandError{Operation: operation, Result: result}
	}
	return nil
}

func (s *DesktopStream) cleanup(ctx context.Context, vncPort, noVNCPID uint32, noVNCCommand string) error {
	var errs []error
	if vncPort != 0 {
		command := "pkill -f " + shellEscape(x11VNCProcessMatch(int(vncPort)))
		if err := s.cleanupCommand(ctx, "stop x11vnc", command); err != nil {
			errs = append(errs, err)
		}
	}
	stopNoVNC := noVNCCommand
	if noVNCPID != 0 {
		stopNoVNC = "kill " + shellEscape(strconv.FormatUint(uint64(noVNCPID), 10))
	}
	if stopNoVNC != "" {
		if err := s.cleanupCommand(ctx, "stop noVNC", stopNoVNC); err != nil {
			errs = append(errs, err)
		}
	}
	if err := s.cleanupCommand(ctx, "remove VNC password", "rm -f ~/.vnc/passwd"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (s *DesktopStream) cleanupAfterStartFailure(primary error, vncPort, noVNCPID uint32, noVNCCommand string) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), desktopCleanupTimeout)
	defer cancel()
	return errors.Join(primary, s.cleanup(cleanupCtx, vncPort, noVNCPID, noVNCCommand))
}

func (s *DesktopStream) storeVNCPassword(ctx context.Context, password string) error {
	command := "IFS= read -r password && x11vnc -storepasswd \"$password\" ~/.vnc/passwd"
	options := append(s.desktop.commandOptions(), WithStdin())
	handle, err := s.desktop.commands.Start(ctx, command, options...)
	if err != nil {
		return fmt.Errorf("desktop start VNC password helper: %w", err)
	}
	stopHelper := func() {
		handle.Disconnect()
		if handle.commands != nil {
			killCtx, cancel := context.WithTimeout(context.Background(), desktopCleanupTimeout)
			defer cancel()
			_ = handle.Kill(killCtx)
		}
	}
	pid, err := handle.WaitPID(ctx)
	if err != nil {
		stopHelper()
		return fmt.Errorf("desktop VNC password helper PID: %w", err)
	}
	if err := s.desktop.commands.SendStdin(ctx, pid, []byte(password+"\n")); err != nil {
		stopHelper()
		return fmt.Errorf("desktop send VNC password: %w", err)
	}
	result, err := handle.WaitContext(ctx)
	handle.Disconnect()
	if err != nil {
		stopHelper()
		return fmt.Errorf("desktop store VNC password: %w", err)
	}
	if result.ExitCode != 0 {
		return &DesktopCommandError{Operation: "store VNC password", Result: result}
	}
	return nil
}

// Start 启动 x11vnc 和 noVNC。默认启用随机 VNC 密码认证；已启动时返回错误。
func (s *DesktopStream) Start(ctx context.Context, options *DesktopStreamOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("desktop stream is already running")
	}
	if s.desktop == nil || s.desktop.Sandbox == nil {
		return fmt.Errorf("desktop sandbox is required")
	}

	normalized, requireAuth, err := normalizeDesktopStreamOptions(options)
	if err != nil {
		return err
	}
	if s.desktop.GetHost(normalized.WebPort) == "" {
		return fmt.Errorf("desktop stream host is unavailable")
	}
	alreadyRunning, err := s.checkRunning(ctx, normalized.VNCPort)
	if err != nil {
		return err
	}
	if alreadyRunning {
		return fmt.Errorf("desktop stream is already running")
	}

	password := ""
	pwdFlag := "-nopw"
	if requireAuth {
		password, err = generateDesktopStreamKey(desktopStreamKeyLength)
		if err != nil {
			return err
		}
		if _, err := s.desktop.run(ctx, "prepare VNC password", "mkdir -p ~/.vnc"); err != nil {
			return err
		}
		if err := s.storeVNCPassword(ctx, password); err != nil {
			return s.cleanupAfterStartFailure(err, 0, 0, "")
		}
		pwdFlag = "-usepw"
	}

	x11Command := fmt.Sprintf(
		"x11vnc -bg -display %s -forever -wait '50' -shared -rfbport %s %s 2>/tmp/x11vnc_stderr.log",
		shellEscape(s.desktop.options.Display), shellEscape(strconv.Itoa(normalized.VNCPort)), pwdFlag,
	)
	if normalized.WindowID != "" {
		x11Command += " -id " + shellEscape(normalized.WindowID)
	}
	if _, err := s.desktop.run(ctx, "start x11vnc", x11Command); err != nil {
		return s.cleanupAfterStartFailure(err, uint32(normalized.VNCPort), 0, "")
	}

	noVNCCommand := fmt.Sprintf(
		"cd /opt/noVNC/utils && exec ./novnc_proxy --vnc %s --listen %s --web /opt/noVNC > /tmp/novnc.log 2>&1",
		shellEscape("localhost:"+strconv.Itoa(normalized.VNCPort)), shellEscape(strconv.Itoa(normalized.WebPort)),
	)
	noVNCProcessMatch := "pkill -f " + shellEscape("[n]ovnc_proxy.*--listen "+strconv.Itoa(normalized.WebPort))
	handle, err := s.desktop.commands.Start(ctx, noVNCCommand, s.desktop.commandOptions()...)
	if err != nil {
		return s.cleanupAfterStartFailure(fmt.Errorf("desktop start noVNC: %w", err), uint32(normalized.VNCPort), 0, noVNCProcessMatch)
	}
	noVNCPID, err := handle.WaitPID(ctx)
	handle.Disconnect()
	if err != nil {
		return s.cleanupAfterStartFailure(fmt.Errorf("desktop start noVNC PID: %w", err), uint32(normalized.VNCPort), noVNCPID, noVNCProcessMatch)
	}

	err = s.desktop.waitUntil(ctx, defaultDesktopStreamTimeout, func(ctx context.Context) (bool, error) {
		command := "netstat -tuln | grep -F -- " + shellEscape(":"+strconv.Itoa(normalized.WebPort)+" ")
		result, probeErr := s.desktop.commands.Run(ctx, command, s.desktop.commandOptions()...)
		if probeErr != nil {
			return false, fmt.Errorf("desktop noVNC readiness probe: %w", probeErr)
		}
		return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "", nil
	})
	if err != nil {
		return s.cleanupAfterStartFailure(fmt.Errorf("wait for noVNC: %w", err), uint32(normalized.VNCPort), noVNCPID, noVNCProcessMatch)
	}

	s.running = true
	s.vncPort = normalized.VNCPort
	s.webPort = normalized.WebPort
	s.password = password
	s.noVNCPID = noVNCPID
	s.requirePW = requireAuth
	return nil
}

// Stop 停止 x11vnc 和 noVNC。未启动时调用是安全的；即使部分清理失败，
// Stop 也会清除本地运行状态和内存中的认证密码，调用方应检查返回错误。
func (s *DesktopStream) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	err := s.cleanup(ctx, uint32(s.vncPort), s.noVNCPID, "")
	s.running = false
	s.vncPort = 0
	s.webPort = 0
	s.password = ""
	s.noVNCPID = 0
	s.requirePW = false
	return err
}

// AuthKey 返回当前流的 VNC 密码。未运行或未启用认证时返回错误。
func (s *DesktopStream) AuthKey() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return "", fmt.Errorf("desktop stream is not running")
	}
	if !s.requirePW || s.password == "" {
		return "", fmt.Errorf("desktop stream authentication is disabled")
	}
	return s.password, nil
}

// URL 返回 noVNC 页面地址。AuthKey 仅在调用方显式提供时加入查询参数；
// URL 中的凭据可能被浏览器历史、代理和日志记录。
func (s *DesktopStream) URL(options *DesktopStreamURLOptions) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return "", fmt.Errorf("desktop stream is not running")
	}

	autoConnect := true
	viewOnly := false
	resize := DesktopStreamResizeScale
	authKey := ""
	if options != nil {
		if options.AutoConnect != nil {
			autoConnect = *options.AutoConnect
		}
		viewOnly = options.ViewOnly
		if options.Resize != "" {
			resize = options.Resize
		}
		authKey = options.AuthKey
	}
	if resize != DesktopStreamResizeOff && resize != DesktopStreamResizeScale && resize != DesktopStreamResizeRemote {
		return "", fmt.Errorf("invalid desktop stream resize mode %q", resize)
	}

	streamURL := url.URL{
		Scheme: "https",
		Host:   s.desktop.GetHost(s.webPort),
		Path:   "/vnc.html",
	}
	query := streamURL.Query()
	if autoConnect {
		query.Set("autoconnect", "true")
	}
	if viewOnly {
		query.Set("view_only", "true")
	}
	query.Set("resize", string(resize))
	if authKey != "" {
		query.Set("password", authKey)
	}
	streamURL.RawQuery = query.Encode()
	return streamURL.String(), nil
}
