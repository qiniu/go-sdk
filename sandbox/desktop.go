package sandbox

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultDesktopTemplateID 是 Desktop Sandbox 使用的默认模板 ID。
	DefaultDesktopTemplateID = "desktop"

	defaultDesktopWidth       = 1024
	defaultDesktopHeight      = 768
	defaultDesktopDPI         = 96
	defaultDesktopDisplay     = ":0"
	defaultDesktopProbePeriod = 500 * time.Millisecond
	defaultXvfbStartTimeout   = 10 * time.Second
	defaultXfceStartTimeout   = 60 * time.Second
	defaultTypeTextChunkSize  = 25
	defaultTypeTextDelay      = 75 * time.Millisecond
	desktopCleanupTimeout     = 10 * time.Second
)

var (
	displayPattern        = regexp.MustCompile(`^[A-Za-z0-9_.-]*:[0-9]+(?:\.[0-9]+)?$`)
	screenCurrentPattern  = regexp.MustCompile(`current\s+(\d+)\s+x\s+(\d+)`)
	cursorPositionPattern = regexp.MustCompile(`x:(\d+)\s+y:(\d+)`)
	keyPattern            = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

// ScreenSize 表示桌面分辨率。
type ScreenSize struct {
	Width  int
	Height int
}

// Point 表示桌面坐标。
type Point struct {
	X int
	Y int
}

// DesktopOptions 是桌面环境配置。
type DesktopOptions struct {
	Resolution ScreenSize
	DPI        int
	Display    string
}

// DesktopCreateParams 是创建 Desktop Sandbox 的参数。
type DesktopCreateParams struct {
	CreateParams
	Resolution ScreenSize
	DPI        int
	Display    string
}

// MouseButton 表示鼠标按钮。
type MouseButton string

const (
	// MouseButtonLeft 表示鼠标左键。
	MouseButtonLeft MouseButton = "left"
	// MouseButtonMiddle 表示鼠标中键。
	MouseButtonMiddle MouseButton = "middle"
	// MouseButtonRight 表示鼠标右键。
	MouseButtonRight MouseButton = "right"
)

// ScrollDirection 表示鼠标滚轮方向。
type ScrollDirection string

const (
	// ScrollDirectionUp 表示向上滚动。
	ScrollDirectionUp ScrollDirection = "up"
	// ScrollDirectionDown 表示向下滚动。
	ScrollDirectionDown ScrollDirection = "down"
)

// TypeTextOptions 是文本输入选项。
type TypeTextOptions struct {
	// ChunkSize 是每次输入的 Unicode 字符数；零值使用默认值。
	ChunkSize int
	// Delay 是 xdotool 输入每个字符时的间隔；零值使用默认值。
	Delay time.Duration
}

// DesktopCommandError 表示桌面命令以非零状态退出。
type DesktopCommandError struct {
	Operation string
	Result    *CommandResult
}

// Error 实现 error 接口。
func (e *DesktopCommandError) Error() string {
	if e == nil || e.Result == nil {
		return "desktop command failed"
	}
	if e.Result.Stderr != "" {
		return fmt.Sprintf("desktop %s failed with exit code %d: %s", e.Operation, e.Result.ExitCode, strings.TrimSpace(e.Result.Stderr))
	}
	return fmt.Sprintf("desktop %s failed with exit code %d", e.Operation, e.Result.ExitCode)
}

type desktopCommandRunner interface {
	Run(context.Context, string, ...CommandOption) (*CommandResult, error)
	Start(context.Context, string, ...CommandOption) (*CommandHandle, error)
	SendStdin(context.Context, uint32, []byte) error
}

type desktopFilesystem interface {
	Read(context.Context, string, ...FilesystemOption) ([]byte, error)
	Remove(context.Context, string, ...FilesystemOption) error
}

// Desktop 提供 Desktop Sandbox 的屏幕、鼠标和键盘操作。
type Desktop struct {
	*Sandbox

	options  DesktopOptions
	commands desktopCommandRunner
	files    desktopFilesystem

	startMu    sync.Mutex
	started    bool
	actionMu   sync.Mutex
	streamOnce sync.Once
	stream     *DesktopStream
}

func defaultDesktopOptions() DesktopOptions {
	return DesktopOptions{
		Resolution: ScreenSize{Width: defaultDesktopWidth, Height: defaultDesktopHeight},
		DPI:        defaultDesktopDPI,
		Display:    defaultDesktopDisplay,
	}
}

func normalizeDesktopOptions(options DesktopOptions) (DesktopOptions, error) {
	defaults := defaultDesktopOptions()
	if options.Resolution == (ScreenSize{}) {
		options.Resolution = defaults.Resolution
	}
	if options.DPI == 0 {
		options.DPI = defaults.DPI
	}
	if options.Display == "" {
		options.Display = defaults.Display
	}
	if options.Resolution.Width <= 0 || options.Resolution.Height <= 0 {
		return DesktopOptions{}, fmt.Errorf("desktop resolution must be positive")
	}
	if options.DPI <= 0 {
		return DesktopOptions{}, fmt.Errorf("desktop DPI must be positive")
	}
	if !displayPattern.MatchString(options.Display) {
		return DesktopOptions{}, fmt.Errorf("invalid desktop display %q", options.Display)
	}
	return options, nil
}

func normalizeDesktopCreateParams(params DesktopCreateParams) (CreateParams, DesktopOptions, error) {
	options, err := normalizeDesktopOptions(DesktopOptions{
		Resolution: params.Resolution,
		DPI:        params.DPI,
		Display:    params.Display,
	})
	if err != nil {
		return CreateParams{}, DesktopOptions{}, err
	}

	createParams := params.CreateParams
	if createParams.TemplateID == "" {
		createParams.TemplateID = DefaultDesktopTemplateID
	}
	envs := make(map[string]string)
	if createParams.EnvVars != nil {
		for key, value := range *createParams.EnvVars {
			envs[key] = value
		}
	}
	envs["DISPLAY"] = options.Display
	createParams.EnvVars = &envs
	return createParams, options, nil
}

func newDesktop(sb *Sandbox, options DesktopOptions) *Desktop {
	return &Desktop{
		Sandbox:  sb,
		options:  options,
		commands: sb.Commands(),
		files:    sb.Files(),
	}
}

// NewDesktop 将已有的 Sandbox 包装为 Desktop。该函数不启动桌面环境；
// 调用方应按需调用 [Desktop.Start]。
func NewDesktop(sb *Sandbox, options DesktopOptions) (*Desktop, error) {
	if sb == nil {
		return nil, fmt.Errorf("sandbox is required")
	}
	normalized, err := normalizeDesktopOptions(options)
	if err != nil {
		return nil, err
	}
	return newDesktop(sb, normalized), nil
}

// Stream 返回当前桌面的 noVNC 流管理器。
func (d *Desktop) Stream() *DesktopStream {
	d.streamOnce.Do(func() {
		d.stream = newDesktopStream(d)
	})
	return d.stream
}

// Display 返回桌面使用的 X11 display。
func (d *Desktop) Display() string {
	return d.options.Display
}

type desktopStarter func(context.Context, *Sandbox, DesktopOptions) (*Desktop, error)

// CreateDesktop 创建并初始化 Desktop Sandbox。
// TemplateID 为空时使用 [DefaultDesktopTemplateID]。创建后的任一步骤失败时，
// SDK 会尽力终止已经创建的 Sandbox。
func (c *Client) CreateDesktop(ctx context.Context, params DesktopCreateParams, pollOpts ...PollOption) (*Desktop, error) {
	return c.createDesktop(ctx, params, func(ctx context.Context, sb *Sandbox, options DesktopOptions) (*Desktop, error) {
		desktop := newDesktop(sb, options)
		if err := desktop.Start(ctx); err != nil {
			return nil, err
		}
		return desktop, nil
	}, pollOpts...)
}

func (c *Client) createDesktop(ctx context.Context, params DesktopCreateParams, starter desktopStarter, pollOpts ...PollOption) (*Desktop, error) {
	createParams, options, err := normalizeDesktopCreateParams(params)
	if err != nil {
		return nil, err
	}
	sb, err := c.Create(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("create desktop sandbox: %w", err)
	}
	cleanup := func(primary error) error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), desktopCleanupTimeout)
		defer cancel()
		return errors.Join(primary, sb.Kill(cleanupCtx))
	}
	if _, err := c.WaitForReady(ctx, sb.ID(), pollOpts...); err != nil {
		return nil, cleanup(fmt.Errorf("wait for desktop sandbox: %w", err))
	}
	desktop, err := starter(ctx, sb, options)
	if err != nil {
		return nil, cleanup(fmt.Errorf("initialize desktop sandbox: %w", err))
	}
	return desktop, nil
}

func (d *Desktop) commandOptions() []CommandOption {
	return []CommandOption{WithEnvs(map[string]string{"DISPLAY": d.options.Display})}
}

func (d *Desktop) run(ctx context.Context, operation, command string) (*CommandResult, error) {
	result, err := d.commands.Run(ctx, command, d.commandOptions()...)
	if err != nil {
		return nil, fmt.Errorf("desktop %s: %w", operation, err)
	}
	if result.ExitCode != 0 {
		return nil, &DesktopCommandError{Operation: operation, Result: result}
	}
	return result, nil
}

func (d *Desktop) startProcess(ctx context.Context, operation, command string) error {
	handle, err := d.commands.Start(ctx, command, d.commandOptions()...)
	if err != nil {
		return fmt.Errorf("desktop %s: %w", operation, err)
	}
	if _, err := handle.WaitPID(ctx); err != nil {
		return fmt.Errorf("desktop %s PID: %w", operation, err)
	}
	handle.Disconnect()
	return nil
}

func (d *Desktop) waitUntil(ctx context.Context, timeout time.Duration, probe func(context.Context) (bool, error)) error {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		ready, err := probe(probeCtx)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		timer := time.NewTimer(defaultDesktopProbePeriod)
		select {
		case <-timer.C:
		case <-probeCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return probeCtx.Err()
		}
	}
}

func (d *Desktop) probeCommand(ctx context.Context, command string) (bool, error) {
	result, err := d.commands.Run(ctx, command, d.commandOptions()...)
	if err != nil {
		return false, fmt.Errorf("desktop readiness probe: %w", err)
	}
	return result.ExitCode == 0, nil
}

// Start 启动 Xvfb 和 Xfce 桌面环境，并等待其就绪。
func (d *Desktop) Start(ctx context.Context) error {
	d.startMu.Lock()
	defer d.startMu.Unlock()
	if d.started {
		return nil
	}

	resolution := fmt.Sprintf("%dx%dx24", d.options.Resolution.Width, d.options.Resolution.Height)
	// -ac is safe here because TCP transport is disabled and the Unix socket is
	// only reachable by processes inside this single-tenant Sandbox.
	xvfbCommand := fmt.Sprintf(
		"Xvfb %s -ac -screen 0 %s -retro -dpi %s -nolisten tcp",
		shellEscape(d.options.Display), shellEscape(resolution), shellEscape(strconv.Itoa(d.options.DPI)),
	)
	if err := d.startProcess(ctx, "start Xvfb", xvfbCommand); err != nil {
		return err
	}
	if err := d.waitUntil(ctx, defaultXvfbStartTimeout, func(ctx context.Context) (bool, error) {
		return d.probeCommand(ctx, "xdpyinfo -display "+shellEscape(d.options.Display))
	}); err != nil {
		return fmt.Errorf("wait for Xvfb: %w", err)
	}

	if err := d.startProcess(ctx, "start Xfce", "startxfce4"); err != nil {
		return err
	}
	if err := d.waitUntil(ctx, defaultXfceStartTimeout, func(ctx context.Context) (bool, error) {
		return d.probeCommand(ctx, "pgrep -x xfce4-session >/dev/null && pgrep -x xfwm4 >/dev/null && pgrep -x xfdesktop >/dev/null")
	}); err != nil {
		return fmt.Errorf("wait for Xfce: %w", err)
	}

	d.started = true
	return nil
}

// ScreenSize 返回当前桌面分辨率。
func (d *Desktop) ScreenSize(ctx context.Context) (ScreenSize, error) {
	result, err := d.run(ctx, "get screen size", "xrandr")
	if err != nil {
		return ScreenSize{}, err
	}
	match := screenCurrentPattern.FindStringSubmatch(result.Stdout)
	if match == nil {
		return ScreenSize{}, fmt.Errorf("parse desktop screen size from %q", result.Stdout)
	}
	width, widthErr := strconv.Atoi(match[1])
	height, heightErr := strconv.Atoi(match[2])
	if widthErr != nil || heightErr != nil {
		return ScreenSize{}, fmt.Errorf("parse desktop screen size %q", match[0])
	}
	return ScreenSize{Width: width, Height: height}, nil
}

// CursorPosition 返回当前鼠标坐标。
func (d *Desktop) CursorPosition(ctx context.Context) (Point, error) {
	result, err := d.run(ctx, "get cursor position", "xdotool getmouselocation")
	if err != nil {
		return Point{}, err
	}
	match := cursorPositionPattern.FindStringSubmatch(result.Stdout)
	if match == nil {
		return Point{}, fmt.Errorf("parse desktop cursor position from %q", result.Stdout)
	}
	x, xErr := strconv.Atoi(match[1])
	y, yErr := strconv.Atoi(match[2])
	if xErr != nil || yErr != nil {
		return Point{}, fmt.Errorf("parse desktop cursor position %q", match[0])
	}
	return Point{X: x, Y: y}, nil
}

func validatePoint(point Point) error {
	if point.X < 0 || point.Y < 0 {
		return fmt.Errorf("desktop coordinates must be non-negative")
	}
	return nil
}

func (d *Desktop) moveMouse(ctx context.Context, point Point) error {
	if err := validatePoint(point); err != nil {
		return err
	}
	_, err := d.run(ctx, "move mouse", fmt.Sprintf(
		"xdotool mousemove --sync %s %s",
		shellEscape(strconv.Itoa(point.X)), shellEscape(strconv.Itoa(point.Y)),
	))
	return err
}

// MoveMouse 将鼠标移动到指定坐标。
func (d *Desktop) MoveMouse(ctx context.Context, point Point) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	return d.moveMouse(ctx, point)
}

func mouseButtonNumber(button MouseButton) (string, error) {
	switch button {
	case MouseButtonLeft:
		return "1", nil
	case MouseButtonMiddle:
		return "2", nil
	case MouseButtonRight:
		return "3", nil
	default:
		return "", fmt.Errorf("invalid mouse button %q", button)
	}
}

// Click 在当前位置或指定坐标单击鼠标按钮。point 为 nil 时使用当前位置。
func (d *Desktop) Click(ctx context.Context, button MouseButton, point *Point) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	if point != nil {
		if err := d.moveMouse(ctx, *point); err != nil {
			return err
		}
	}
	buttonNumber, err := mouseButtonNumber(button)
	if err != nil {
		return err
	}
	_, err = d.run(ctx, "click mouse", "xdotool click "+shellEscape(buttonNumber))
	return err
}

// DoubleClick 在当前位置或指定坐标双击鼠标左键。point 为 nil 时使用当前位置。
func (d *Desktop) DoubleClick(ctx context.Context, point *Point) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	if point != nil {
		if err := d.moveMouse(ctx, *point); err != nil {
			return err
		}
	}
	_, err := d.run(ctx, "double click mouse", "xdotool click --repeat '2' '1'")
	return err
}

func (d *Desktop) mouseButtonAction(ctx context.Context, operation, action string, button MouseButton) error {
	buttonNumber, err := mouseButtonNumber(button)
	if err != nil {
		return err
	}
	_, err = d.run(ctx, operation, "xdotool "+action+" "+shellEscape(buttonNumber))
	return err
}

// MouseDown 按下指定鼠标按钮但不释放。
func (d *Desktop) MouseDown(ctx context.Context, button MouseButton) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	return d.mouseButtonAction(ctx, "press mouse button", "mousedown", button)
}

// MouseUp 释放指定鼠标按钮。
func (d *Desktop) MouseUp(ctx context.Context, button MouseButton) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	return d.mouseButtonAction(ctx, "release mouse button", "mouseup", button)
}

// Drag 按住鼠标左键从起点拖动到终点。整个操作期间会串行化其他输入操作。
func (d *Desktop) Drag(ctx context.Context, from, to Point) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	if err := d.moveMouse(ctx, from); err != nil {
		return err
	}
	if err := d.mouseButtonAction(ctx, "press mouse button", "mousedown", MouseButtonLeft); err != nil {
		return err
	}
	if err := d.moveMouse(ctx, to); err != nil {
		return err
	}
	return d.mouseButtonAction(ctx, "release mouse button", "mouseup", MouseButtonLeft)
}

// Scroll 按指定方向滚动鼠标滚轮。
func (d *Desktop) Scroll(ctx context.Context, direction ScrollDirection, amount int) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	if amount <= 0 {
		return fmt.Errorf("desktop scroll amount must be positive")
	}
	button := ""
	switch direction {
	case ScrollDirectionUp:
		button = "4"
	case ScrollDirectionDown:
		button = "5"
	default:
		return fmt.Errorf("invalid desktop scroll direction %q", direction)
	}
	command := fmt.Sprintf("xdotool click --repeat %s %s", shellEscape(strconv.Itoa(amount)), shellEscape(button))
	_, err := d.run(ctx, "scroll mouse", command)
	return err
}

// TypeText 在当前光标位置输入文本。整个操作期间会串行化其他输入操作；
// 对长文本可通过 TypeTextOptions.ChunkSize 控制远程命令次数。
func (d *Desktop) TypeText(ctx context.Context, text string, options *TypeTextOptions) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	chunkSize := defaultTypeTextChunkSize
	delay := defaultTypeTextDelay
	if options != nil {
		if options.ChunkSize != 0 {
			chunkSize = options.ChunkSize
		}
		if options.Delay != 0 {
			delay = options.Delay
		}
	}
	if chunkSize <= 0 {
		return fmt.Errorf("desktop text chunk size must be positive")
	}
	if delay < 0 {
		return fmt.Errorf("desktop text delay must be non-negative")
	}
	runes := []rune(text)
	for start := 0; start < len(runes); start += chunkSize {
		end := min(start+chunkSize, len(runes))
		command := fmt.Sprintf(
			"xdotool type --delay %s -- %s",
			shellEscape(strconv.FormatInt(delay.Milliseconds(), 10)), shellEscape(string(runes[start:end])),
		)
		if _, err := d.run(ctx, "type text", command); err != nil {
			return err
		}
	}
	return nil
}

var desktopKeyNames = map[string]string{
	"alt": "Alt_L", "alt_left": "Alt_L", "alt_right": "Alt_R",
	"backspace": "BackSpace", "break": "Pause", "caps_lock": "Caps_Lock",
	"cmd": "Super_L", "command": "Super_L", "control": "Control_L",
	"control_left": "Control_L", "control_right": "Control_R", "ctrl": "Control_L",
	"del": "Delete", "delete": "Delete", "down": "Down", "end": "End",
	"enter": "Return", "esc": "Escape", "escape": "Escape",
	"f1": "F1", "f2": "F2", "f3": "F3", "f4": "F4", "f5": "F5", "f6": "F6",
	"f7": "F7", "f8": "F8", "f9": "F9", "f10": "F10", "f11": "F11", "f12": "F12",
	"home": "Home", "insert": "Insert", "left": "Left", "menu": "Menu", "meta": "Meta_L",
	"num_lock": "Num_Lock", "page_down": "Page_Down", "page_up": "Page_Up", "pause": "Pause",
	"print": "Print", "right": "Right", "scroll_lock": "Scroll_Lock", "shift": "Shift_L",
	"shift_left": "Shift_L", "shift_right": "Shift_R", "space": "space", "super": "Super_L",
	"super_left": "Super_L", "super_right": "Super_R", "tab": "Tab", "up": "Up",
	"win": "Super_L", "windows": "Super_L",
}

// Press 按下单个按键或组合键。按键名称只能包含 ASCII 字母、数字和下划线，
// 可用名称及组合键格式以 xdotool 支持的名称为准。
func (d *Desktop) Press(ctx context.Context, keys ...string) error {
	d.actionMu.Lock()
	defer d.actionMu.Unlock()
	if len(keys) == 0 {
		return fmt.Errorf("at least one desktop key is required")
	}
	mapped := make([]string, len(keys))
	for i, key := range keys {
		if !keyPattern.MatchString(key) {
			return fmt.Errorf("invalid key %q", key)
		}
		lower := strings.ToLower(key)
		if value, ok := desktopKeyNames[lower]; ok {
			mapped[i] = value
		} else {
			mapped[i] = lower
		}
	}
	_, err := d.run(ctx, "press key", "xdotool key "+shellEscape(strings.Join(mapped, "+")))
	return err
}

// Open 使用默认桌面应用打开文件或 URL。
func (d *Desktop) Open(ctx context.Context, pathOrURL string) error {
	if pathOrURL == "" {
		return fmt.Errorf("desktop path or URL is required")
	}
	return d.startProcess(ctx, "open file or URL", "xdg-open "+shellEscape(pathOrURL))
}

// Launch 通过 gtk-launch 启动桌面应用。
func (d *Desktop) Launch(ctx context.Context, application string, args ...string) error {
	if application == "" {
		return fmt.Errorf("desktop application is required")
	}
	parts := make([]string, 0, len(args)+2)
	parts = append(parts, "gtk-launch", shellEscape(application))
	for _, arg := range args {
		parts = append(parts, shellEscape(arg))
	}
	return d.startProcess(ctx, "launch application", strings.Join(parts, " "))
}

// ActiveWindowID 返回当前活动窗口 ID。
func (d *Desktop) ActiveWindowID(ctx context.Context) (string, error) {
	result, err := d.run(ctx, "get active window", "xdotool getwindowfocus")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ApplicationWindows 返回指定应用所有可见窗口的 ID。
func (d *Desktop) ApplicationWindows(ctx context.Context, application string) ([]string, error) {
	if application == "" {
		return nil, fmt.Errorf("desktop application is required")
	}
	result, err := d.commands.Run(
		ctx,
		"xdotool search --onlyvisible --class "+shellEscape(application),
		d.commandOptions()...,
	)
	if err != nil {
		return nil, fmt.Errorf("desktop find application windows: %w", err)
	}
	if result.ExitCode == 1 && strings.TrimSpace(result.Stdout) == "" {
		return []string{}, nil
	}
	if result.ExitCode != 0 {
		return nil, &DesktopCommandError{Operation: "find application windows", Result: result}
	}
	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

// WindowTitle 返回指定窗口的标题。
func (d *Desktop) WindowTitle(ctx context.Context, windowID string) (string, error) {
	if windowID == "" {
		return "", fmt.Errorf("desktop window ID is required")
	}
	result, err := d.run(ctx, "get window title", "xdotool getwindowname "+shellEscape(windowID))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// Wait 等待指定时长，ctx 取消时提前返回。
func (d *Desktop) Wait(ctx context.Context, duration time.Duration) error {
	if duration < 0 {
		return fmt.Errorf("desktop wait duration must be non-negative")
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Screenshot 获取包含鼠标指针的 PNG 截图。
func (d *Desktop) Screenshot(ctx context.Context) ([]byte, error) {
	path := "/tmp/qiniu-desktop-screenshot-" + uuid.NewString() + ".png"
	if _, err := d.run(ctx, "take screenshot", "scrot --pointer "+shellEscape(path)); err != nil {
		return nil, err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), desktopCleanupTimeout)
		defer cancel()
		_ = d.files.Remove(cleanupCtx, path)
	}()
	data, err := d.files.Read(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read desktop screenshot: %w", err)
	}
	return data, nil
}
