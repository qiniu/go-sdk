//go:build unit

package sandbox

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

type desktopCommandCall struct {
	cmd  string
	opts *commandOpts
}

type fakeDesktopCommands struct {
	runResults []*CommandResult
	runErr     error
	runCalls   []desktopCommandCall
	startCalls []desktopCommandCall
	disconnect int
}

func (f *fakeDesktopCommands) Run(_ context.Context, cmd string, opts ...CommandOption) (*CommandResult, error) {
	f.runCalls = append(f.runCalls, desktopCommandCall{cmd: cmd, opts: applyCommandOpts(opts)})
	if f.runErr != nil {
		return nil, f.runErr
	}
	if len(f.runResults) == 0 {
		return &CommandResult{}, nil
	}
	result := f.runResults[0]
	f.runResults = f.runResults[1:]
	return result, nil
}

func (f *fakeDesktopCommands) Start(_ context.Context, cmd string, opts ...CommandOption) (*CommandHandle, error) {
	f.startCalls = append(f.startCalls, desktopCommandCall{cmd: cmd, opts: applyCommandOpts(opts)})
	handle := &CommandHandle{
		cancel: func() { f.disconnect++ },
		done:   make(chan struct{}),
		pidCh:  make(chan struct{}),
	}
	handle.markPIDReady(uint32(100 + len(f.startCalls)))
	return handle, nil
}

type fakeDesktopFiles struct {
	data        []byte
	readPath    string
	removedPath string
}

func (f *fakeDesktopFiles) Read(_ context.Context, path string, _ ...FilesystemOption) ([]byte, error) {
	f.readPath = path
	return f.data, nil
}

func (f *fakeDesktopFiles) Remove(_ context.Context, path string, _ ...FilesystemOption) error {
	f.removedPath = path
	return nil
}

func testDesktop(commands desktopCommandRunner, files desktopFilesystem) *Desktop {
	return &Desktop{
		options:  defaultDesktopOptions(),
		commands: commands,
		files:    files,
	}
}

func TestNormalizeDesktopCreateParamsUsesDefaultsWithoutMutatingEnv(t *testing.T) {
	envs := map[string]string{"EXISTING": "value", "DISPLAY": ":9"}
	params := DesktopCreateParams{CreateParams: CreateParams{EnvVars: &envs}}

	normalized, options, err := normalizeDesktopCreateParams(params)
	if err != nil {
		t.Fatalf("normalizeDesktopCreateParams error: %v", err)
	}

	if normalized.TemplateID != DefaultDesktopTemplateID {
		t.Fatalf("TemplateID = %q, want %q", normalized.TemplateID, DefaultDesktopTemplateID)
	}
	if options.Resolution != (ScreenSize{Width: 1024, Height: 768}) {
		t.Fatalf("Resolution = %+v", options.Resolution)
	}
	if options.DPI != 96 || options.Display != ":0" {
		t.Fatalf("options = %+v", options)
	}
	if got := (*normalized.EnvVars)["DISPLAY"]; got != ":0" {
		t.Fatalf("normalized DISPLAY = %q, want :0", got)
	}
	if got := envs["DISPLAY"]; got != ":9" {
		t.Fatalf("input env map was mutated: DISPLAY = %q", got)
	}
}

func TestNewDesktopWrapsExistingSandboxWithDefaults(t *testing.T) {
	domain := "sandbox.test"
	sb := &Sandbox{
		sandboxID: "desktop-1",
		domain:    &domain,
		client:    &Client{config: &Config{HTTPClient: http.DefaultClient}},
	}
	desktop, err := NewDesktop(sb, DesktopOptions{})
	if err != nil {
		t.Fatalf("NewDesktop error: %v", err)
	}
	if desktop.Sandbox != sb {
		t.Fatal("NewDesktop did not preserve Sandbox")
	}
	if desktop.options != defaultDesktopOptions() {
		t.Fatalf("Desktop options = %+v", desktop.options)
	}
	if desktop.Display() != defaultDesktopDisplay {
		t.Fatalf("Display = %q", desktop.Display())
	}
	if _, err := NewDesktop(nil, DesktopOptions{}); err == nil {
		t.Fatal("NewDesktop accepted nil Sandbox")
	}
}

func TestDesktopStartLaunchesXvfbAndXfce(t *testing.T) {
	commands := &fakeDesktopCommands{runResults: []*CommandResult{{ExitCode: 0}, {ExitCode: 0}}}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	if err := desktop.Start(context.Background()); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	if len(commands.startCalls) != 2 {
		t.Fatalf("Start calls = %d, want 2", len(commands.startCalls))
	}
	if got := commands.startCalls[0].cmd; got != "Xvfb ':0' -ac -screen 0 '1024x768x24' -retro -dpi '96' -nolisten tcp" {
		t.Fatalf("Xvfb command = %q", got)
	}
	if got := commands.startCalls[1].cmd; got != "startxfce4" {
		t.Fatalf("Xfce command = %q", got)
	}
	if commands.disconnect != 2 {
		t.Fatalf("Disconnect calls = %d, want 2", commands.disconnect)
	}
	for _, call := range commands.runCalls {
		if got := call.opts.envs["DISPLAY"]; got != ":0" {
			t.Fatalf("probe DISPLAY = %q, want :0", got)
		}
	}
}

func TestDesktopScreenAndCursorParsing(t *testing.T) {
	commands := &fakeDesktopCommands{runResults: []*CommandResult{
		{ExitCode: 0, Stdout: "Screen 0: minimum 8 x 8, current 1440 x 900, maximum 32767 x 32767\n"},
		{ExitCode: 0, Stdout: "x:0 y:899 screen:0 window:123\n"},
	}}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	size, err := desktop.ScreenSize(context.Background())
	if err != nil {
		t.Fatalf("ScreenSize error: %v", err)
	}
	if size != (ScreenSize{Width: 1440, Height: 900}) {
		t.Fatalf("ScreenSize = %+v", size)
	}
	point, err := desktop.CursorPosition(context.Background())
	if err != nil {
		t.Fatalf("CursorPosition error: %v", err)
	}
	if point != (Point{X: 0, Y: 899}) {
		t.Fatalf("CursorPosition = %+v", point)
	}
}

func TestDesktopClickAcceptsOriginAndSerializesArguments(t *testing.T) {
	commands := &fakeDesktopCommands{}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	point := Point{X: 0, Y: 0}
	if err := desktop.Click(context.Background(), MouseButtonLeft, &point); err != nil {
		t.Fatalf("Click error: %v", err)
	}

	want := []string{
		"xdotool mousemove --sync '0' '0'",
		"xdotool click '1'",
	}
	if len(commands.runCalls) != len(want) {
		t.Fatalf("Run calls = %d, want %d", len(commands.runCalls), len(want))
	}
	for i, call := range commands.runCalls {
		if call.cmd != want[i] {
			t.Fatalf("Run call %d = %q, want %q", i, call.cmd, want[i])
		}
	}
}

func TestDesktopTypeTextShellEscapesEachUnicodeChunk(t *testing.T) {
	commands := &fakeDesktopCommands{}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	err := desktop.TypeText(context.Background(), "你好'a", &TypeTextOptions{
		ChunkSize: 2,
		Delay:     5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("TypeText error: %v", err)
	}

	want := []string{
		"xdotool type --delay '5' -- '你好'",
		"xdotool type --delay '5' -- ''\"'\"'a'",
	}
	if len(commands.runCalls) != len(want) {
		t.Fatalf("Run calls = %d, want %d", len(commands.runCalls), len(want))
	}
	for i, call := range commands.runCalls {
		if call.cmd != want[i] {
			t.Fatalf("Run call %d = %q, want %q", i, call.cmd, want[i])
		}
	}
}

func TestDesktopPressRejectsShellSyntax(t *testing.T) {
	desktop := testDesktop(&fakeDesktopCommands{}, &fakeDesktopFiles{})

	err := desktop.Press(context.Background(), "ctrl", "c; touch /tmp/injected")
	if err == nil || !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("Press error = %v, want invalid key", err)
	}
}

func TestDesktopScreenshotReadsAndRemovesTemporaryFile(t *testing.T) {
	commands := &fakeDesktopCommands{}
	files := &fakeDesktopFiles{data: []byte("png")}
	desktop := testDesktop(commands, files)

	data, err := desktop.Screenshot(context.Background())
	if err != nil {
		t.Fatalf("Screenshot error: %v", err)
	}
	if string(data) != "png" {
		t.Fatalf("Screenshot = %q", data)
	}
	if !strings.HasPrefix(files.readPath, "/tmp/qiniu-desktop-screenshot-") || !strings.HasSuffix(files.readPath, ".png") {
		t.Fatalf("Read path = %q", files.readPath)
	}
	if files.removedPath != files.readPath {
		t.Fatalf("Removed path = %q, want %q", files.removedPath, files.readPath)
	}
}

func TestDesktopCommandFailureReturnsTypedError(t *testing.T) {
	commands := &fakeDesktopCommands{runResults: []*CommandResult{{ExitCode: 2, Stderr: "bad input"}}}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	_, err := desktop.ScreenSize(context.Background())
	var commandErr *DesktopCommandError
	if !errors.As(err, &commandErr) {
		t.Fatalf("ScreenSize error = %T %v, want *DesktopCommandError", err, err)
	}
	if commandErr.Result.ExitCode != 2 || commandErr.Result.Stderr != "bad input" {
		t.Fatalf("DesktopCommandError result = %+v", commandErr.Result)
	}
}

func TestDesktopDragKeepsMouseSequenceTogether(t *testing.T) {
	commands := &fakeDesktopCommands{}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	if err := desktop.Drag(context.Background(), Point{X: 1, Y: 2}, Point{X: 3, Y: 4}); err != nil {
		t.Fatalf("Drag error: %v", err)
	}

	want := []string{
		"xdotool mousemove --sync '1' '2'",
		"xdotool mousedown '1'",
		"xdotool mousemove --sync '3' '4'",
		"xdotool mouseup '1'",
	}
	if len(commands.runCalls) != len(want) {
		t.Fatalf("Run calls = %d, want %d", len(commands.runCalls), len(want))
	}
	for i, call := range commands.runCalls {
		if call.cmd != want[i] {
			t.Fatalf("Run call %d = %q, want %q", i, call.cmd, want[i])
		}
	}
}

func TestDesktopDoubleClickAndScroll(t *testing.T) {
	commands := &fakeDesktopCommands{}
	desktop := testDesktop(commands, &fakeDesktopFiles{})
	point := Point{X: 8, Y: 9}

	if err := desktop.DoubleClick(context.Background(), &point); err != nil {
		t.Fatalf("DoubleClick error: %v", err)
	}
	if err := desktop.Scroll(context.Background(), ScrollDirectionUp, 3); err != nil {
		t.Fatalf("Scroll error: %v", err)
	}

	wantLast := "xdotool click --repeat '3' '4'"
	if got := commands.runCalls[len(commands.runCalls)-1].cmd; got != wantLast {
		t.Fatalf("Scroll command = %q, want %q", got, wantLast)
	}
	if err := desktop.Scroll(context.Background(), ScrollDirection("sideways"), 1); err == nil {
		t.Fatal("Scroll accepted an invalid direction")
	}
}

func TestDesktopOpenAndLaunchStartDetachedCommands(t *testing.T) {
	commands := &fakeDesktopCommands{}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	if err := desktop.Open(context.Background(), "https://example.com/a?x='value'"); err != nil {
		t.Fatalf("Open error: %v", err)
	}
	if err := desktop.Launch(context.Background(), "firefox-esr", "--new-window", "https://example.com"); err != nil {
		t.Fatalf("Launch error: %v", err)
	}

	want := []string{
		"xdg-open 'https://example.com/a?x='\"'\"'value'\"'\"''",
		"gtk-launch 'firefox-esr' '--new-window' 'https://example.com'",
	}
	if len(commands.startCalls) != len(want) {
		t.Fatalf("Start calls = %d, want %d", len(commands.startCalls), len(want))
	}
	for i, call := range commands.startCalls {
		if call.cmd != want[i] {
			t.Fatalf("Start call %d = %q, want %q", i, call.cmd, want[i])
		}
	}
	if commands.disconnect != 2 {
		t.Fatalf("Disconnect calls = %d, want 2", commands.disconnect)
	}
}

func TestDesktopWindowQueries(t *testing.T) {
	commands := &fakeDesktopCommands{runResults: []*CommandResult{
		{ExitCode: 0, Stdout: "42\n"},
		{ExitCode: 0, Stdout: "42\n84\n"},
		{ExitCode: 0, Stdout: "Example Domain\n"},
	}}
	desktop := testDesktop(commands, &fakeDesktopFiles{})

	active, err := desktop.ActiveWindowID(context.Background())
	if err != nil || active != "42" {
		t.Fatalf("ActiveWindowID = %q, %v", active, err)
	}
	windows, err := desktop.ApplicationWindows(context.Background(), "firefox-esr")
	if err != nil || len(windows) != 2 || windows[1] != "84" {
		t.Fatalf("ApplicationWindows = %v, %v", windows, err)
	}
	title, err := desktop.WindowTitle(context.Background(), "42")
	if err != nil || title != "Example Domain" {
		t.Fatalf("WindowTitle = %q, %v", title, err)
	}
}

func TestDesktopWaitHonorsContextCancellation(t *testing.T) {
	desktop := testDesktop(&fakeDesktopCommands{}, &fakeDesktopFiles{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := desktop.Wait(ctx, time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait error = %v, want context.Canceled", err)
	}
}

func TestCreateDesktopCleansUpSandboxWhenDesktopStartupFails(t *testing.T) {
	startupErr := errors.New("xfce did not start")
	accessToken := "envd-token"
	deleted := false
	var gotBody apis.CreateSandboxJSONRequestBody
	mock := &mockAPI{
		createSandboxFn: func(_ context.Context, _ *apis.CreateSandboxParams, body apis.CreateSandboxJSONRequestBody, _ ...apis.RequestEditorFn) (*apis.CreateSandboxResponse, error) {
			gotBody = body
			return &apis.CreateSandboxResponse{
				JSON201: &apis.Sandbox{
					SandboxID:       "desktop-1",
					TemplateID:      DefaultDesktopTemplateID,
					EnvdAccessToken: &accessToken,
				},
				HTTPResponse: httpResponse(201),
			}, nil
		},
		getSandboxFn: func(_ context.Context, sandboxID apis.SandboxID, _ ...apis.RequestEditorFn) (*apis.GetSandboxResponse, error) {
			return &apis.GetSandboxResponse{
				JSON200:      &apis.SandboxDetail{SandboxID: sandboxID, State: apis.Running},
				HTTPResponse: httpResponse(200),
			}, nil
		},
		deleteSandboxFn: func(_ context.Context, sandboxID apis.SandboxID, _ ...apis.RequestEditorFn) (*apis.DeleteSandboxResponse, error) {
			if sandboxID != "desktop-1" {
				t.Fatalf("deleted sandbox ID = %q", sandboxID)
			}
			deleted = true
			return &apis.DeleteSandboxResponse{HTTPResponse: httpResponse(204)}, nil
		},
	}
	client := newTestClient(mock)

	_, err := client.createDesktop(
		context.Background(),
		DesktopCreateParams{},
		func(context.Context, *Sandbox, DesktopOptions) (*Desktop, error) { return nil, startupErr },
	)
	if !errors.Is(err, startupErr) {
		t.Fatalf("createDesktop error = %v, want startup error", err)
	}
	if !deleted {
		t.Fatal("desktop startup failure did not delete the sandbox")
	}
	if gotBody.TemplateID != DefaultDesktopTemplateID {
		t.Fatalf("TemplateID = %q", gotBody.TemplateID)
	}
	if gotBody.EnvVars == nil || (*gotBody.EnvVars)["DISPLAY"] != defaultDesktopDisplay {
		t.Fatalf("EnvVars = %v", gotBody.EnvVars)
	}
}
