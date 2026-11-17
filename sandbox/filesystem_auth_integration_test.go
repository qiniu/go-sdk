//go:build integration

package sandbox

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationSecuredFilesystemHTTP(t *testing.T) {
	client := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	templates, err := client.ListTemplates(ctx, nil)
	require.NoError(t, err)
	var templateID string
	for _, template := range templates {
		if template.BuildStatus == BuildStatusReady || template.BuildStatus == BuildStatusUploaded {
			templateID = template.TemplateID
			break
		}
	}
	if templateID == "" {
		t.Skip("没有可用模板，跳过测试")
	}
	secure, timeout := true, int32(120)
	sb, err := client.Create(ctx, CreateParams{TemplateID: templateID, Secure: &secure, Timeout: &timeout})
	require.NoError(t, err)
	t.Cleanup(func() {
		killCtx, killCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer killCancel()
		if err := sb.Kill(killCtx); err != nil {
			t.Logf("清理沙箱 %s 失败: %v", sb.ID(), err)
		}
	})
	_, err = sb.WaitForReady(ctx)
	require.NoError(t, err)
	require.NotNil(t, sb.envdAccessToken)
	require.NotEmpty(t, *sb.envdAccessToken)
	_, err = sb.Files().Write(ctx, "/tmp/sdk-secure-single.txt", []byte("single"))
	require.NoError(t, err)
	_, err = sb.Files().WriteFiles(ctx, []WriteEntry{
		{Path: "/tmp/sdk-secure-a.txt", Data: []byte("batch-a")},
		{Path: "/tmp/sdk-secure-b.txt", Data: []byte("batch-b")},
	})
	require.NoError(t, err)
	for path, want := range map[string]string{
		"/tmp/sdk-secure-single.txt": "single",
		"/tmp/sdk-secure-a.txt":      "batch-a",
		"/tmp/sdk-secure-b.txt":      "batch-b",
	} {
		content, err := sb.Files().ReadText(ctx, path)
		require.NoError(t, err)
		assert.Equal(t, want, content)
	}

	// 使用独立 HTTP 客户端，不附加 token，验证公开 URL 本身的签名。
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sb.DownloadURL("/tmp/sdk-secure-single.txt"), nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "single", string(content))
}
