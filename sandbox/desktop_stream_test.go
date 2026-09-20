//go:build unit

package sandbox

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

type fakeDesktopStreamCommands struct {
	runResults    []*CommandResult
	runCalls      []desktopCommandCall
	startCalls    []desktopCommandCall
	startErr      error
	waitPID       bool
	disconnect    int
	stdinCalls    int
	cancelOnStdin context.CancelFunc
}

func (f *fakeDesktopStreamCommands) Run(_ context.Context, cmd string, opts ...CommandOption) (*CommandResult, error) {
	f.runCalls = append(f.runCalls, desktopCommandCall{cmd: cmd, opts: applyCommandOpts(opts)})
	if len(f.runResults) == 0 {
		return &CommandResult{}, nil
	}
	result := f.runResults[0]
	f.runResults = f.runResults[1:]
	return result, nil
}

func (f *fakeDesktopStreamCommands) Start(_ context.Context, cmd string, opts ...CommandOption) (*CommandHandle, error) {
	f.startCalls = append(f.startCalls, desktopCommandCall{cmd: cmd, opts: applyCommandOpts(opts)})
	if f.startErr != nil && strings.Contains(cmd, "novnc_proxy") {
		return nil, f.startErr
	}
	handle := &CommandHandle{
		cancel: func() { f.disconnect++ },
		done:   make(chan struct{}),
		pidCh:  make(chan struct{}),
	}
	if strings.Contains(cmd, "storepasswd") {
		handle.result = &CommandResult{}
		close(handle.done)
	}
	if !f.waitPID || strings.Contains(cmd, "storepasswd") {
		handle.markPIDReady(321)
	}
	return handle, nil
}

func (f *fakeDesktopStreamCommands) SendStdin(context.Context, uint32, []byte) error {
	f.stdinCalls++
	if f.cancelOnStdin != nil {
		f.cancelOnStdin()
	}
	return nil
}

func (f *fakeDesktopStreamCommands) CloseStdin(context.Context, uint32) error { return nil }

func newStreamTestDesktop(commands desktopCommandRunner) *Desktop {
	domain := "sandbox.test"
	return &Desktop{
		Sandbox:  &Sandbox{sandboxID: "desktop-1", domain: &domain},
		options:  defaultDesktopOptions(),
		commands: commands,
		files:    &fakeDesktopFiles{},
	}
}

func TestDesktopStreamStartUsesSecureDefaultsAndBuildsURL(t *testing.T) {
	commands := &fakeDesktopStreamCommands{runResults: []*CommandResult{
		{ExitCode: 1},
		{ExitCode: 0},
		{ExitCode: 0},
		{ExitCode: 0},
		{ExitCode: 0, Stdout: "tcp 0 0 0.0.0.0:6080 0.0.0.0:* LISTEN\n"},
	}}
	stream := newStreamTestDesktop(commands).Stream()

	if err := stream.Start(context.Background(), nil); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	key, err := stream.AuthKey()
	if err != nil {
		t.Fatalf("AuthKey error: %v", err)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{16}$`).MatchString(key) {
		t.Fatalf("AuthKey = %q, want 16 alphanumeric characters", key)
	}
	if commands.disconnect != 2 {
		t.Fatalf("Disconnect calls = %d, want 2", commands.disconnect)
	}
	if len(commands.startCalls) != 2 || !strings.Contains(commands.startCalls[1].cmd, "--vnc 'localhost:5900' --listen '6080'") {
		t.Fatalf("noVNC start calls = %+v", commands.startCalls)
	}
	joined := ""
	for _, call := range commands.startCalls {
		joined += call.cmd + "\n"
	}
	if !strings.Contains(joined, "IFS= read -r password && x11vnc -storepasswd \"$password\"") {
		t.Fatalf("storepasswd command does not read password from stdin: %s", joined)
	}
	if commands.stdinCalls != 1 {
		t.Fatalf("SendStdin calls = %d, want 1", commands.stdinCalls)
	}
	joined = ""
	for _, call := range commands.runCalls {
		joined += call.cmd + "\n"
	}
	if !strings.Contains(joined, "-rfbport '5900' -usepw") {
		t.Fatalf("x11vnc command does not use secure defaults: %s", joined)
	}

	rawURL, err := stream.URL(&DesktopStreamURLOptions{
		ViewOnly: true,
		AuthKey:  "p&=# value",
	})
	if err != nil {
		t.Fatalf("URL error: %v", err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "6080-desktop-1.sandbox.test" || parsed.Path != "/vnc.html" {
		t.Fatalf("URL = %q", rawURL)
	}
	query := parsed.Query()
	if query.Get("autoconnect") != "true" || query.Get("view_only") != "true" || query.Get("resize") != "scale" {
		t.Fatalf("URL query = %v", query)
	}
	if query.Get("password") != "p&=# value" {
		t.Fatalf("URL password = %q", query.Get("password"))
	}

	before := len(commands.runCalls)
	if err := stream.Start(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("second Start error = %v, want already running", err)
	}
	if len(commands.runCalls) != before {
		t.Fatal("second Start executed commands")
	}
}

func TestDesktopStreamStopTerminatesBothProcessesAndClearsSecrets(t *testing.T) {
	commands := &fakeDesktopStreamCommands{runResults: []*CommandResult{
		{ExitCode: 1}, {ExitCode: 0}, {ExitCode: 0}, {ExitCode: 0},
		{ExitCode: 0, Stdout: "tcp 0 0 0.0.0.0:6080 0.0.0.0:* LISTEN\n"},
	}}
	stream := newStreamTestDesktop(commands).Stream()
	if err := stream.Start(context.Background(), nil); err != nil {
		t.Fatalf("Start error: %v", err)
	}

	if err := stream.Stop(context.Background()); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	joined := ""
	for _, call := range commands.runCalls {
		joined += call.cmd + "\n"
	}
	if !strings.Contains(joined, "pkill -f '[x]11vnc.*-rfbport 5900'") ||
		!strings.Contains(joined, "kill '321'") ||
		!strings.Contains(joined, "rm -f ~/.vnc/passwd") {
		t.Fatalf("cleanup commands = %s", joined)
	}
	if _, err := stream.URL(nil); err == nil {
		t.Fatal("URL succeeded after Stop")
	}
	if _, err := stream.AuthKey(); err == nil {
		t.Fatal("AuthKey succeeded after Stop")
	}
	if err := stream.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop error: %v", err)
	}
}

func TestDesktopStreamStopClearsStateAfterCleanupError(t *testing.T) {
	commands := &fakeDesktopStreamCommands{runResults: []*CommandResult{
		{ExitCode: 1}, {ExitCode: 0}, {ExitCode: 0},
		{ExitCode: 0, Stdout: "tcp 0 0 0.0.0.0:6080 0.0.0.0:* LISTEN\n"},
		{ExitCode: 2, Stderr: "permission denied"}, {ExitCode: 0}, {ExitCode: 0},
	}}
	stream := newStreamTestDesktop(commands).Stream()
	if err := stream.Start(context.Background(), nil); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := stream.Stop(context.Background()); err == nil {
		t.Fatal("Stop succeeded despite cleanup failure")
	}
	if _, err := stream.AuthKey(); err == nil {
		t.Fatal("AuthKey succeeded after failed Stop cleanup")
	}
	if _, err := stream.URL(nil); err == nil {
		t.Fatal("URL succeeded after failed Stop cleanup")
	}
}

func TestDesktopStreamCleansUpX11VNCWhenNoVNCStartFails(t *testing.T) {
	startErr := errors.New("envd stream failed")
	commands := &fakeDesktopStreamCommands{
		runResults: []*CommandResult{{ExitCode: 1}, {ExitCode: 0}, {ExitCode: 0}, {ExitCode: 0}},
		startErr:   startErr,
	}
	stream := newStreamTestDesktop(commands).Stream()

	err := stream.Start(context.Background(), nil)
	if !errors.Is(err, startErr) {
		t.Fatalf("Start error = %v, want %v", err, startErr)
	}
	joined := ""
	for _, call := range commands.runCalls {
		joined += call.cmd + "\n"
	}
	if !strings.Contains(joined, "pkill -f '[x]11vnc.*-rfbport 5900'") || !strings.Contains(joined, "rm -f ~/.vnc/passwd") {
		t.Fatalf("cleanup commands = %s", joined)
	}
	if _, err := stream.URL(nil); err == nil {
		t.Fatal("URL succeeded after failed Start")
	}
}

func TestDesktopStreamCleansUpNoVNCByPortWhenPIDDiscoveryIsCanceled(t *testing.T) {
	commands := &fakeDesktopStreamCommands{
		runResults: []*CommandResult{
			{ExitCode: 1},
			{ExitCode: 0},
			{ExitCode: 0},
			{ExitCode: 0},
			{ExitCode: 0},
			{ExitCode: 0},
			{ExitCode: 0},
		},
		waitPID: true,
	}
	stream := newStreamTestDesktop(commands).Stream()
	ctx, cancel := context.WithCancel(context.Background())
	commands.cancelOnStdin = cancel

	err := stream.Start(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start error = %v, want context.Canceled", err)
	}
	joined := ""
	for _, call := range commands.runCalls {
		joined += call.cmd + "\n"
	}
	if !strings.Contains(joined, "pkill -f '[n]ovnc_proxy.*--listen 6080'") {
		t.Fatalf("noVNC process cleanup command missing: %s", joined)
	}
	if !strings.Contains(joined, "rm -f ~/.vnc/passwd") {
		t.Fatalf("password cleanup command missing: %s", joined)
	}
}

func TestDesktopStreamCanDisableVNCAuthentication(t *testing.T) {
	no := false
	commands := &fakeDesktopStreamCommands{runResults: []*CommandResult{
		{ExitCode: 1}, {ExitCode: 0},
		{ExitCode: 0, Stdout: "tcp 0 0 0.0.0.0:6080 0.0.0.0:* LISTEN\n"},
	}}
	stream := newStreamTestDesktop(commands).Stream()

	if err := stream.Start(context.Background(), &DesktopStreamOptions{RequireAuth: &no}); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if _, err := stream.AuthKey(); err == nil {
		t.Fatal("AuthKey succeeded with authentication disabled")
	}
	joined := ""
	for _, call := range commands.runCalls {
		joined += call.cmd + "\n"
	}
	if strings.Contains(joined, "storepasswd") || !strings.Contains(joined, "-nopw") {
		t.Fatalf("authentication-disabled commands = %s", joined)
	}
}

func TestDesktopStreamUsesCustomPortsAndWindowID(t *testing.T) {
	commands := &fakeDesktopStreamCommands{runResults: []*CommandResult{
		{ExitCode: 1}, {ExitCode: 0},
		{ExitCode: 0, Stdout: "tcp 0 0 0.0.0.0:6081 0.0.0.0:* LISTEN\n"},
	}}
	stream := newStreamTestDesktop(commands).Stream()
	if err := stream.Start(context.Background(), &DesktopStreamOptions{
		VNCPort: 5901,
		WebPort: 6081,
		RequireAuth: func() *bool {
			value := false
			return &value
		}(),
		WindowID: "42",
	}); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if len(commands.startCalls) != 1 || !strings.Contains(commands.startCalls[0].cmd, "--listen '6081'") {
		t.Fatalf("noVNC start command = %+v", commands.startCalls)
	}
	joined := ""
	for _, call := range commands.runCalls {
		joined += call.cmd + "\n"
	}
	if !strings.Contains(joined, "-rfbport '5901' -nopw") || !strings.Contains(joined, "-id '42'") {
		t.Fatalf("x11vnc custom command = %s", joined)
	}
	autoConnect := false
	rawURL, err := stream.URL(&DesktopStreamURLOptions{AutoConnect: &autoConnect, Resize: DesktopStreamResizeOff})
	if err != nil {
		t.Fatalf("URL error: %v", err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if parsed.Host != "6081-desktop-1.sandbox.test" || parsed.Query().Get("autoconnect") != "" || parsed.Query().Get("resize") != "off" {
		t.Fatalf("custom URL = %q", rawURL)
	}
	if err := stream.Stop(context.Background()); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
}

func TestNormalizeDesktopStreamOptionsRejectsInvalidPorts(t *testing.T) {
	for _, options := range []*DesktopStreamOptions{
		{VNCPort: 0, WebPort: 0},
		{VNCPort: -1},
		{WebPort: 65536},
		{VNCPort: 5900, WebPort: 5900},
	} {
		if options.VNCPort == 0 && options.WebPort == 0 {
			continue
		}
		if _, _, err := normalizeDesktopStreamOptions(options); err == nil {
			t.Fatalf("normalizeDesktopStreamOptions accepted %+v", options)
		}
	}
}
