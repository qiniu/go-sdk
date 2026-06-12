package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/sandbox"
)

func main() {
	loadEnvFileIfExists("examples/sandbox_resources/.env", ".env")

	// 确保设置了环境变量 QINIU_API_KEY
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 QINIU_API_KEY 环境变量")
	}

	apiURL := strings.TrimSpace(os.Getenv("QINIU_SANDBOX_API_URL"))

	ctx := context.Background()
	gitConfig := loadGitResourceConfig()
	kodoConfig, err := loadKodoResourceConfig()
	if err != nil {
		log.Fatal(err)
	}

	if !gitConfig.Enabled() && !kodoConfig.Enabled() {
		log.Fatal("请设置 GITHUB_TOKEN 和 QINIU_SANDBOX_GIT_REPO_URL，或设置 QINIU_SANDBOX_KODO_BUCKET，以启用至少一种资源")
	}

	var failures []error
	if gitConfig.Enabled() {
		client, err := sandbox.NewClient(&sandbox.Config{
			APIKey:   apiKey,
			Endpoint: apiURL,
		})
		if err != nil {
			failures = append(failures, fmt.Errorf("create Git resource client: %w", err))
		} else if err := runGitResourceExample(ctx, client, gitConfig); err != nil {
			failures = append(failures, fmt.Errorf("Git resource example: %w", err))
			log.Printf("Git resource example failed: %v", err)
		}
	}

	if kodoConfig.Enabled() {
		if kodoConfig.AccessKey == "" || kodoConfig.SecretKey == "" {
			failures = append(failures, errors.New("启用 Kodo 资源时请设置 QINIU_ACCESS_KEY 和 QINIU_SECRET_KEY 环境变量"))
		} else {
			client, err := sandbox.NewClient(&sandbox.Config{
				APIKey:      apiKey,
				Credentials: auth.New(kodoConfig.AccessKey, kodoConfig.SecretKey),
				Endpoint:    apiURL,
			})
			if err != nil {
				failures = append(failures, fmt.Errorf("create Kodo resource client: %w", err))
			} else if err := runKodoResourceExample(ctx, client, kodoConfig); err != nil {
				failures = append(failures, fmt.Errorf("Kodo resource example: %w", err))
				log.Printf("Kodo resource example failed: %v", err)
			}
		}
	}

	if len(failures) > 0 {
		log.Fatal(errors.Join(failures...))
	}
}

type gitResourceConfig struct {
	RepoURL       string
	Token         string
	MountPath     string
	PushTest      bool
	TestFile      string
	CommitName    string
	CommitEmail   string
	CommitMessage string
	PushBranch    string
}

func loadGitResourceConfig() gitResourceConfig {
	return gitResourceConfig{
		RepoURL:       os.Getenv("QINIU_SANDBOX_GIT_REPO_URL"),
		Token:         os.Getenv("GITHUB_TOKEN"),
		MountPath:     "/workspace/repo",
		PushTest:      envBool("QINIU_SANDBOX_GIT_PUSH_TEST", false),
		TestFile:      envDefault("QINIU_SANDBOX_GIT_TEST_FILE", "sandbox-resource-write-test.txt"),
		CommitName:    envDefault("QINIU_SANDBOX_GIT_COMMIT_NAME", "Sandbox Resource Demo"),
		CommitEmail:   envDefault("QINIU_SANDBOX_GIT_COMMIT_EMAIL", "sandbox-resource-demo@example.com"),
		CommitMessage: envDefault("QINIU_SANDBOX_GIT_COMMIT_MESSAGE", "test: update sandbox resource write file"),
		PushBranch:    os.Getenv("QINIU_SANDBOX_GIT_PUSH_BRANCH"),
	}
}

func (c gitResourceConfig) Enabled() bool {
	return c.Token != "" && c.RepoURL != ""
}

func runGitResourceExample(ctx context.Context, client *sandbox.Client, cfg gitResourceConfig) error {
	resource := sandbox.GitRepositoryResource{
		Type:               sandbox.GitRepositoryTypeGithub,
		URL:                cfg.RepoURL,
		MountPath:          cfg.MountPath,
		AuthorizationToken: &cfg.Token,
	}
	resources := []sandbox.SandboxResourceSpec{{GitRepository: &resource}}

	log.Println("Creating sandbox with Git repository resource...")
	sb, info, err := client.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: "base",
		Resources:  &resources,
	})
	if err != nil {
		return err
	}
	defer func() {
		log.Println("Killing Git resource sandbox...")
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = sb.Kill(cleanupCtx)
	}()

	log.Printf("Git resource sandbox created successfully! ID: %s, State: %s\n", sb.ID(), info.State)
	if err := runAndLog(ctx, sb, "ls -la "+shellQuote(cfg.MountPath)+" | head -20"); err != nil {
		return err
	}

	var pushURL string
	if cfg.PushTest {
		pushURL, err = githubTokenPushURL(cfg.RepoURL)
		if err != nil {
			return err
		}
		if err := runAndLog(ctx, sb, gitSyncBranchCommand(cfg.MountPath, cfg.PushBranch, pushURL), sandbox.WithTimeout(2*time.Minute), sandbox.WithEnvs(map[string]string{
			"GITHUB_TOKEN":        cfg.Token,
			"GIT_TERMINAL_PROMPT": "0",
		})); err != nil {
			return err
		}
	}

	writeContent := "sandbox resource write test\n" +
		"time: " + time.Now().UTC().Format(time.RFC3339Nano) + "\n" +
		"sandbox: " + sb.ID() + "\n"
	gitFilePath := path.Join(cfg.MountPath, cfg.TestFile)
	if err := runAndLog(ctx, sb, writeFileCommand(gitFilePath, writeContent+"resource: git\n")); err != nil {
		return err
	}
	if err := runAndLog(ctx, sb, "git -C "+shellQuote(cfg.MountPath)+" status --short "+shellQuote(cfg.TestFile)); err != nil {
		return err
	}

	if cfg.PushTest {
		if err := runAndLog(ctx, sb, gitCommitAndPushCommand(cfg.MountPath, cfg.TestFile, cfg.CommitName, cfg.CommitEmail, cfg.CommitMessage, cfg.PushBranch), sandbox.WithTimeout(2*time.Minute), sandbox.WithEnvs(map[string]string{
			"GITHUB_TOKEN":        cfg.Token,
			"GIT_TERMINAL_PROMPT": "0",
		})); err != nil {
			return err
		}
	}
	return nil
}

type kodoResourceConfig struct {
	Bucket    string
	MountPath string
	Prefix    string
	ReadOnly  *bool
	AccessKey string
	SecretKey string
	WriteTest bool
	TestFile  string
}

func loadKodoResourceConfig() (kodoResourceConfig, error) {
	readOnly, err := optionalEnvBool("QINIU_SANDBOX_KODO_READ_ONLY")
	if err != nil {
		return kodoResourceConfig{}, err
	}
	return kodoResourceConfig{
		Bucket:    os.Getenv("QINIU_SANDBOX_KODO_BUCKET"),
		MountPath: envDefault("QINIU_SANDBOX_KODO_MOUNT_PATH", "/workspace/kodo"),
		Prefix:    os.Getenv("QINIU_SANDBOX_KODO_PREFIX"),
		ReadOnly:  readOnly,
		AccessKey: os.Getenv("QINIU_ACCESS_KEY"),
		SecretKey: os.Getenv("QINIU_SECRET_KEY"),
		WriteTest: envBool("QINIU_SANDBOX_KODO_WRITE_TEST", true),
		TestFile:  envDefault("QINIU_SANDBOX_KODO_TEST_FILE", "sandbox-resource-write-test.txt"),
	}, nil
}

func (c kodoResourceConfig) Enabled() bool {
	return c.Bucket != ""
}

func runKodoResourceExample(ctx context.Context, client *sandbox.Client, cfg kodoResourceConfig) error {
	resource := sandbox.KodoResource{
		Bucket:    cfg.Bucket,
		MountPath: cfg.MountPath,
	}
	if cfg.Prefix != "" {
		resource.Prefix = &cfg.Prefix
	}
	if cfg.ReadOnly != nil {
		resource.ReadOnly = cfg.ReadOnly
	}
	resources := []sandbox.SandboxResourceSpec{{Kodo: &resource}}

	log.Println("Creating sandbox with Kodo bucket resource...")
	sb, info, err := client.CreateAndWait(ctx, sandbox.CreateParams{
		TemplateID: "base",
		Resources:  &resources,
	})
	if err != nil {
		return err
	}
	defer func() {
		log.Println("Killing Kodo resource sandbox...")
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = sb.Kill(cleanupCtx)
	}()

	log.Printf("Kodo resource sandbox created successfully! ID: %s, State: %s\n", sb.ID(), info.State)
	if err := runAndLog(ctx, sb, "ls -la "+shellQuote(cfg.MountPath)+" | head -20"); err != nil {
		return err
	}

	if cfg.WriteTest {
		if cfg.ReadOnlyEnabled() {
			log.Println("Skipping Kodo write test because QINIU_SANDBOX_KODO_READ_ONLY=true")
			return nil
		}
		writeContent := "sandbox resource write test\n" +
			"time: " + time.Now().UTC().Format(time.RFC3339Nano) + "\n" +
			"sandbox: " + sb.ID() + "\n" +
			"resource: kodo\n"
		if err := runAndLog(ctx, sb, writeFileCommand(path.Join(cfg.MountPath, cfg.TestFile), writeContent)); err != nil {
			return err
		}
	}
	return nil
}

func (c kodoResourceConfig) ReadOnlyEnabled() bool {
	return c.ReadOnly != nil && *c.ReadOnly
}

func loadEnvFileIfExists(paths ...string) {
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			value = strings.TrimSpace(value)
			if _, exists := os.LookupEnv(key); key != "" && !exists {
				_ = os.Setenv(key, value)
			}
		}
		if err := file.Close(); err != nil {
			log.Fatalf("Failed to close .env file %s: %v", p, err)
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Failed to read .env file %s: %v", p, err)
		}
		log.Printf("Loaded environment from %s", p)
		return
	}
}

func envDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func optionalEnvBool(key string) (*bool, error) {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch value {
	case "":
		return nil, nil
	case "1", "true", "yes", "on":
		v := true
		return &v, nil
	case "0", "false", "no", "off":
		v := false
		return &v, nil
	default:
		return nil, fmt.Errorf("%s must be one of 1, true, yes, on, 0, false, no, or off", key)
	}
}

func runAndLog(ctx context.Context, sb *sandbox.Sandbox, cmd string, opts ...sandbox.CommandOption) error {
	log.Printf("Executing command in sandbox:\n$ %s\n", cmd)

	result, err := sb.Commands().Run(ctx, cmd, opts...)
	if err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	log.Printf("ExitCode: %d\n", result.ExitCode)
	if result.Stdout != "" {
		log.Printf("Stdout:\n%s\n", result.Stdout)
	}
	if result.Stderr != "" {
		log.Printf("Stderr:\n%s\n", result.Stderr)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}
	return nil
}

func githubTokenPushURL(repoURL string) (string, error) {
	repoURL = strings.TrimSpace(repoURL)
	repoURL = strings.TrimSuffix(repoURL, "/")
	switch {
	case strings.HasPrefix(repoURL, "https://"):
		return "https://x-access-token:${GITHUB_TOKEN}@" + strings.TrimPrefix(repoURL, "https://"), nil
	case strings.HasPrefix(repoURL, "http://"):
		return "https://x-access-token:${GITHUB_TOKEN}@" + strings.TrimPrefix(repoURL, "http://"), nil
	case strings.HasPrefix(repoURL, "git@github.com:"):
		return "https://x-access-token:${GITHUB_TOKEN}@github.com/" + strings.TrimPrefix(repoURL, "git@github.com:"), nil
	default:
		return "", fmt.Errorf("unsupported GitHub repository URL: %s", repoURL)
	}
}

func gitSyncBranchCommand(repoPath, branch, pushURL string) string {
	branchExpr := "branch=" + shellQuote(branch)
	if branch == "" {
		branchExpr = "branch=$(git -C " + shellQuote(repoPath) + " rev-parse --abbrev-ref HEAD)"
	}
	return strings.Join([]string{
		"set -e",
		branchExpr,
		"test \"$branch\" != HEAD",
		"git -C " + shellQuote(repoPath) + " remote set-url origin " + shellDoubleQuote(pushURL),
		"git -C " + shellQuote(repoPath) + " fetch origin \"$branch\"",
		"git -C " + shellQuote(repoPath) + " checkout \"$branch\"",
		"git -C " + shellQuote(repoPath) + " reset --hard \"origin/$branch\"",
	}, " && ")
}

func gitCommitAndPushCommand(repoPath, filePath, name, email, message, branch string) string {
	branchExpr := "branch=" + shellQuote(branch)
	if branch == "" {
		branchExpr = "branch=$(git -C " + shellQuote(repoPath) + " rev-parse --abbrev-ref HEAD)"
	}
	return strings.Join([]string{
		"set -e",
		branchExpr,
		"test \"$branch\" != HEAD",
		"git -C " + shellQuote(repoPath) + " config user.name " + shellQuote(name),
		"git -C " + shellQuote(repoPath) + " config user.email " + shellQuote(email),
		"git -C " + shellQuote(repoPath) + " add " + shellQuote(filePath),
		"git -C " + shellQuote(repoPath) + " commit -m " + shellQuote(message),
		"for attempt in 1 2 3; do if git -C " + shellQuote(repoPath) + " push origin HEAD:\"$branch\"; then break; fi; if [ \"$attempt\" = 3 ]; then exit 1; fi; sleep $((attempt * 2)); done",
	}, " && ")
}

func writeFileCommand(filePath, content string) string {
	dir := path.Dir(filePath)
	return strings.Join([]string{
		"mkdir -p " + shellQuote(dir),
		"printf %s " + shellQuote(content) + " > " + shellQuote(filePath),
		"ls -la " + shellQuote(filePath),
		"sed -n '1,20p' " + shellQuote(filePath),
	}, " && ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func shellDoubleQuote(s string) string {
	const tokenPlaceholder = "__GITHUB_TOKEN_PLACEHOLDER__"
	s = strings.ReplaceAll(s, "${GITHUB_TOKEN}", tokenPlaceholder)
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
	return `"` + strings.ReplaceAll(replacer.Replace(s), tokenPlaceholder, "${GITHUB_TOKEN}") + `"`
}
