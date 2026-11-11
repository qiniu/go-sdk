//go:build integration

package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCreateTemplateMinDiskMB(t *testing.T) {
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	total, free := int32(10240), int32(5120)
	resp, err := c.CreateTemplate(ctx, CreateTemplateParams{MinDiskMB: &total, MinFreeDiskMB: &free})
	require.NoError(t, err)
	require.NotEmpty(t, resp.TemplateID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		assert.NoError(t, c.DeleteTemplate(cleanupCtx, resp.TemplateID))
	})
	assert.NotEmpty(t, resp.BuildID)
}
