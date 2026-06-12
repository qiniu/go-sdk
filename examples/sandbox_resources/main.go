package main

import (
	"bufio"
	"context"
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
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("%v", r)
		}
	}()

	loadEnvFileIfExists("examples/sandbox_resources/.env", ".env")

	// 确保设置了环境变量 QINIU_API_KEY
	apiKey := os.Getenv("QINIU_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 QINIU_API_KEY 环境变量")
	}

	apiURL := strings.TrimSpace(os.Getenv("QINIU_SANDBOX_API_URL"))

	// 演示用的 GitHub 仓库与 token，可通过环境变量覆盖。
	repoURL := os.Getenv("QINIU_SANDBOX_GIT_REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/qiniu/go-sdk.git"
	}
	githubToken := os.Getenv("GITHUB_TOKEN")
	gitMountPath := "/workspace/repo"
	gitPushTest := envBool("QINIU_SANDBOX_GIT_PUSH_TEST", false)
	gitTestFile := os.Getenv("QINIU_SANDBOX_GIT_TEST_FILE")
	if gitTestFile == "" {
		gitTestFile = "sandbox-resource-write-test.txt"
	}
	gitCommitName := envDefault("QINIU_SANDBOX_GIT_COMMIT_NAME", "Sandbox Resource Demo")
	gitCommitEmail := envDefault("QINIU_SANDBOX_GIT_COMMIT_EMAIL", "sandbox-resource-demo@example.com")
	gitCommitMessage := envDefault("QINIU_SANDBOX_GIT_COMMIT_MESSAGE", "test: update sandbox resource write file")
	gitPushBranch := os.Getenv("QINIU_SANDBOX_GIT_PUSH_BRANCH")

	// 演示用的 Kodo 存储桶资源。设置 QINIU_SANDBOX_KODO_BUCKET 后启用。
	kodoBucket := os.Getenv("QINIU_SANDBOX_KODO_BUCKET")
	kodoMountPath := os.Getenv("QINIU_SANDBOX_KODO_MOUNT_PATH")
	if kodoMountPath == "" {
		kodoMountPath = "/workspace/kodo"
	}
	kodoPrefix := os.Getenv("QINIU_SANDBOX_KODO_PREFIX")
	kodoReadOnly := os.Getenv("QINIU_SANDBOX_KODO_READ_ONLY")
	kodoAccessKey := os.Getenv("QINIU_ACCESS_KEY")
	kodoSecretKey := os.Getenv("QINIU_SECRET_KEY")
	kodoWriteTest := envBool("QINIU_SANDBOX_KODO_WRITE_TEST", true)
	kodoTestFile := os.Getenv("QINIU_SANDBOX_KODO_TEST_FILE")
	if kodoTestFile == "" {
		kodoTestFile = "sandbox-resource-write-test.txt"
	}

	ctx := context.Background()

	var credentials *auth.Credentials
	if kodoBucket != "" {
		if kodoAccessKey == "" || kodoSecretKey == "" {
			log.Fatal("启用 Kodo 资源时请设置 QINIU_ACCESS_KEY 和 QINIU_SECRET_KEY 环境变量")
		}
		credentials = auth.New(kodoAccessKey, kodoSecretKey)
	}

	// 初始化客户端
	client, err := sandbox.NewClient(&sandbox.Config{
		APIKey:      apiKey,
		Credentials: credentials,
		Endpoint:    apiURL,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	resources := make([]sandbox.SandboxResourceSpec, 0, 2)
	pathsToList := make([]string, 0, 2)

	if githubToken != "" {
		// 通过 Resources 在沙箱启动前由平台拉取 GitHub 仓库快照并挂载到 /workspace/repo。
		repoResource := sandbox.GitRepositoryResource{
			Type:               sandbox.GitRepositoryTypeGithub,
			URL:                repoURL,
			MountPath:          gitMountPath,
			AuthorizationToken: &githubToken,
		}
		resources = append(resources, sandbox.SandboxResourceSpec{GitRepository: &repoResource})
		pathsToList = append(pathsToList, repoResource.MountPath)
	}

	if kodoBucket != "" {
		// 通过 Resources 在沙箱启动前将 Kodo 存储桶挂载到指定路径。
		kodoResource := sandbox.KodoResource{
			Bucket:    kodoBucket,
			MountPath: kodoMountPath,
		}
		if kodoPrefix != "" {
			kodoResource.Prefix = &kodoPrefix
		}
		if kodoReadOnly != "" {
			readOnly := kodoReadOnly == "true"
			kodoResource.ReadOnly = &readOnly
		}
		resources = append(resources, sandbox.SandboxResourceSpec{Kodo: &kodoResource})
		pathsToList = append(pathsToList, kodoResource.MountPath)
	}

	if len(resources) == 0 {
		log.Fatal("请设置 GITHUB_TOKEN 或 QINIU_SANDBOX_KODO_BUCKET 以启用至少一种资源")
	}

	params := sandbox.CreateParams{
		TemplateID: "base",
		Resources:  &resources,
	}

	log.Printf("Creating sandbox with %d resource(s)...", len(resources))

	// 创建并等待沙箱就绪
	sb, info, err := client.CreateAndWait(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer func() {
		log.Println("Killing sandbox...")
		_ = sb.Kill(ctx)
	}()

	log.Printf("Sandbox created successfully! ID: %s, State: %s\n", sb.ID(), info.State)

	// 列出已挂载的资源内容，验证资源已就位
	for _, path := range pathsToList {
		listCmd := "ls -la " + path + " | head -20"
		runAndLog(ctx, sb, listCmd)
	}

	writeContent := "sandbox resource write test\n" +
		"time: " + time.Now().UTC().Format(time.RFC3339Nano) + "\n" +
		"sandbox: " + sb.ID() + "\n"

	if githubToken != "" {
		gitFilePath := path.Join(gitMountPath, gitTestFile)
		writeCmd := writeFileCommand(gitFilePath, writeContent+"resource: git\n")
		runAndLog(ctx, sb, writeCmd)
		runAndLog(ctx, sb, "git -C "+shellQuote(gitMountPath)+" status --short "+shellQuote(gitTestFile))
	}

	if kodoBucket != "" && kodoWriteTest {
		kodoFilePath := path.Join(kodoMountPath, kodoTestFile)
		runAndLog(ctx, sb, writeFileCommand(kodoFilePath, writeContent+"resource: kodo\n"))
	}

	if githubToken != "" && gitPushTest {
		pushURL, err := githubTokenPushURL(repoURL)
		if err != nil {
			log.Fatalf("Failed to build Git push URL: %v", err)
		}
		runAndLog(ctx, sb, gitCommitAndPushCommand(gitMountPath, gitTestFile, gitCommitName, gitCommitEmail, gitCommitMessage, gitPushBranch, pushURL), sandbox.WithTimeout(2*time.Minute), sandbox.WithEnvs(map[string]string{
			"GITHUB_TOKEN":        githubToken,
			"GIT_TERMINAL_PROMPT": "0",
		}))
	}
}

func loadEnvFileIfExists(paths ...string) {
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			continue
		}
		defer file.Close()

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
			if key != "" {
				_ = os.Setenv(key, value)
			}
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

func runAndLog(ctx context.Context, sb *sandbox.Sandbox, cmd string, opts ...sandbox.CommandOption) {
	log.Printf("Executing command in sandbox:\n$ %s\n", cmd)

	result, err := sb.Commands().Run(ctx, cmd, opts...)
	if err != nil {
		panic(fmt.Sprintf("Failed to run command: %v", err))
	}

	log.Printf("ExitCode: %d\n", result.ExitCode)
	if result.Stdout != "" {
		log.Printf("Stdout:\n%s\n", result.Stdout)
	}
	if result.Stderr != "" {
		log.Printf("Stderr:\n%s\n", result.Stderr)
	}
	if result.ExitCode != 0 {
		panic(fmt.Sprintf("command failed with exit code %d", result.ExitCode))
	}
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

func gitCommitAndPushCommand(repoPath, filePath, name, email, message, branch, pushURL string) string {
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
		"git -C " + shellQuote(repoPath) + " remote set-url origin " + shellDoubleQuote(pushURL),
		"git -C " + shellQuote(repoPath) + " add " + shellQuote(filePath),
		"git -C " + shellQuote(repoPath) + " commit -m " + shellQuote(message),
		"for attempt in 1 2 3; do git -C " + shellQuote(repoPath) + " push origin HEAD:\"$branch\" && break; if [ \"$attempt\" = 3 ]; then exit 1; fi; sleep $((attempt * 2)); done",
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
