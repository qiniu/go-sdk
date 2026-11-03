//go:build integration

package sandbox

import (
	"context"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIntegrationSnapshotQueries(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	limit := int32(1)
	page, err := c.ListSnapshots(ctx, &ListSnapshotsParams{Limit: &limit})
	require.NoError(t, err)
	if page.NextToken != "" {
		_, err = c.ListSnapshots(ctx, &ListSnapshotsParams{Limit: &limit, NextToken: &page.NextToken})
		require.NoError(t, err)
	}
	if len(page.Items) > 0 {
		_, err = c.ListTemplateTags(ctx, page.Items[0].SnapshotID)
		require.NoError(t, err)
	}
}

func TestIntegrationSnapshotLifecycle(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	templates, err := c.ListDefaultTemplates(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, templates, "需要至少一个可用的默认模板")
	templateID := templates[0].TemplateID
	for _, template := range templates {
		for _, name := range template.Names {
			if name == "base" || strings.HasSuffix(name, "/base") {
				templateID = template.TemplateID
			}
		}
	}
	t.Logf("使用默认模板 %s", templateID)
	var sandboxIDs []string
	var snapshotID string
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()
		for _, id := range sandboxIDs {
			err := c.Kill(cleanupCtx, id)
			if err != nil && !isNotFoundError(err) {
				t.Errorf("清理沙箱 %s: %v", id, err)
			} else {
				t.Logf("已清理沙箱 %s", id)
			}
		}
		if snapshotID != "" {
			err := c.DeleteTemplate(cleanupCtx, snapshotID)
			if err != nil && !isNotFoundError(err) {
				t.Errorf("清理快照 %s: %v", snapshotID, err)
			} else {
				t.Logf("已清理快照 %s", snapshotID)
			}
		}
	})
	timeout := int32(300)
	enabled, memory := true, false
	source, err := c.Create(ctx, CreateParams{TemplateID: templateID, Timeout: &timeout, AutoPause: &enabled, AutoPauseMemory: &enabled, AutoResume: &AutoResumeConfig{Enabled: false}})
	require.NoError(t, err)
	sandboxIDs = append(sandboxIDs, source.ID())
	_, err = source.WaitForReady(ctx)
	require.NoError(t, err)
	t.Logf("源沙箱已就绪 %s", source.ID())
	path := "/home/user/sdk-snapshot-lifecycle.txt"
	payload := []byte("sdk-live-snapshot-" + uuid.NewString())
	_, err = source.Files().Write(ctx, path, payload)
	require.NoError(t, err)
	sourceContent, err := source.Files().Read(ctx, path)
	require.NoError(t, err)
	require.Equal(t, payload, sourceContent)
	name, key := "sdk-live-"+uuid.NewString(), uuid.NewString()
	params := CreateSnapshotParams{Memory: &memory, Name: &name, IdempotencyKey: key}
	t.Logf("创建文件系统快照，幂等键 %s，名称 %s", key, name)
	snapshot, err := source.CreateSnapshot(ctx, params)
	require.NoError(t, err)
	snapshotID = snapshot.SnapshotID
	require.NotEmpty(t, snapshotID)
	t.Logf("快照已创建 %s", snapshotID)
	replay, err := source.CreateSnapshot(ctx, params)
	require.NoError(t, err)
	require.Equal(t, snapshotID, replay.SnapshotID, "同键重试应返回原快照")
	page, err := c.ListSnapshots(ctx, &ListSnapshotsParams{SandboxID: &sandboxIDs[0], Name: &name})
	require.NoError(t, err)
	found := false
	for _, item := range page.Items {
		if item.SnapshotID == snapshotID {
			found = true
		}
	}
	require.True(t, found, "快照应能按来源沙箱和名称查询")
	tags, err := c.ListTemplateTags(ctx, snapshotID)
	require.NoError(t, err)
	t.Logf("快照查询、同键重放和标签查询通过（%d 个标签）", len(tags))
	clone, err := c.Create(ctx, CreateParams{TemplateID: snapshotID, Timeout: &timeout})
	require.NoError(t, err)
	sandboxIDs = append(sandboxIDs, clone.ID())
	_, err = clone.WaitForReady(ctx)
	require.NoError(t, err)
	content, err := clone.Files().Read(ctx, path)
	require.NoError(t, err)
	require.Equal(t, payload, content, "从快照创建的沙箱应保留文件")
	t.Logf("快照启动及文件校验通过 %s", clone.ID())
	for _, keepMemory := range []bool{true, false} {
		beforeBoot, err := clone.Commands().Run(ctx, "cat /proc/sys/kernel/random/boot_id")
		require.NoError(t, err)
		require.Zero(t, beforeBoot.ExitCode)
		require.NoError(t, clone.PauseWithParams(ctx, PauseParams{Memory: &keepMemory}))
		info, err := clone.GetInfo(ctx)
		require.NoError(t, err)
		require.Equal(t, StatePaused, info.State)
		resumed, err := c.Connect(ctx, clone.ID(), ConnectParams{})
		require.NoError(t, err)
		_, err = resumed.WaitForReady(ctx)
		require.NoError(t, err)
		content, err := resumed.Files().Read(ctx, path)
		require.NoError(t, err)
		require.Equal(t, payload, content)
		afterBoot, err := resumed.Commands().Run(ctx, "cat /proc/sys/kernel/random/boot_id")
		require.NoError(t, err)
		require.Zero(t, afterBoot.ExitCode)
		require.NotEmpty(t, strings.TrimSpace(afterBoot.Stdout))
		if keepMemory {
			require.Equal(t, beforeBoot.Stdout, afterBoot.Stdout, "保存内存时应恢复原系统")
		} else {
			require.NotEqual(t, beforeBoot.Stdout, afterBoot.Stdout, "仅保存文件系统时应冷启动")
		}
		clone = resumed
		t.Logf("暂停 memory=%t → v2 Connect 默认超时恢复 → 文件校验通过", keepMemory)
	}
}
