package sandbox

import (
	"context"
	"fmt"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// Attach 获取已有 Sandbox 的连接信息，不恢复暂停实例，也不延长生命周期。
// 每次调用从控制面读取当前地址和 envd 凭证，适用于无状态服务的后续操作。
func (c *Client) Attach(ctx context.Context, sandboxID string) (*Sandbox, error) {
	resp, err := c.api.GetSandboxWithResponse(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	detail := resp.JSON200
	if string(detail.State) != "running" {
		return nil, fmt.Errorf("sandbox is not running")
	}
	return newSandbox(c, &apis.Sandbox{SandboxID: detail.SandboxID, TemplateID: detail.TemplateID, ClientID: detail.ClientID, Alias: detail.Alias, Domain: detail.Domain, EnvdAccessToken: detail.EnvdAccessToken}), nil
}
