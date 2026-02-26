package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// WaitForReady waits for a sandbox to be in running state.
// It polls GetSandbox until the sandbox state is "running" or the context is cancelled.
func (c *Client) WaitForReady(ctx context.Context, sandboxID string, pollInterval time.Duration) (*apis.SandboxDetail, error) {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	for {
		resp, err := c.api.GetSandboxWithResponse(ctx, sandboxID)
		if err != nil {
			return nil, fmt.Errorf("get sandbox %s: %w", sandboxID, err)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("get sandbox %s: status %d", sandboxID, resp.StatusCode())
		}
		if resp.JSON200.State == apis.Running {
			return resp.JSON200, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// CreateAndWait creates a sandbox and waits for it to be ready.
func (c *Client) CreateAndWait(ctx context.Context, body apis.CreateSandboxJSONRequestBody, pollInterval time.Duration) (*apis.SandboxDetail, error) {
	createResp, err := c.api.CreateSandboxWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}
	if createResp.JSON201 == nil {
		return nil, fmt.Errorf("create sandbox: status %d", createResp.StatusCode())
	}
	return c.WaitForReady(ctx, createResp.JSON201.SandboxID, pollInterval)
}

// WaitForBuild waits for a template build to complete.
// It polls GetTemplateBuildStatus until the build status is a terminal state ("ready" or "error").
func (c *Client) WaitForBuild(ctx context.Context, templateID, buildID string, pollInterval time.Duration) (*apis.TemplateBuildInfo, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	for {
		resp, err := c.api.GetTemplateBuildStatusWithResponse(ctx, templateID, buildID, nil)
		if err != nil {
			return nil, fmt.Errorf("get build status %s/%s: %w", templateID, buildID, err)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("get build status %s/%s: status %d", templateID, buildID, resp.StatusCode())
		}
		switch resp.JSON200.Status {
		case apis.TemplateBuildStatusReady:
			return resp.JSON200, nil
		case apis.TemplateBuildStatusError:
			return resp.JSON200, fmt.Errorf("build %s/%s failed", templateID, buildID)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
