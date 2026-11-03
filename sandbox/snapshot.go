package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// CreateSnapshotParams 创建持久快照的请求参数。
type CreateSnapshotParams struct {
	// Memory 是否保存内存，默认 true；false 时从快照创建的沙箱会冷启动。
	Memory *bool
	// Name 快照别名，可包含标签，但不能包含斜杠；命名空间由服务端分配。
	Name *string
	// IdempotencyKey 幂等键（必填，最长 128 字节）。每次新捕获都必须使用新键，包括同名快照的新版本；
	// 对结果不确定的同一次捕获，使用原键和原请求体重试。服务端在捕获完成后保留该键 35 分钟，过期后重用可能发起新的捕获。
	IdempotencyKey string
}

// SnapshotInfo 可用于创建沙箱的持久快照模板。
type SnapshotInfo struct {
	// SnapshotID 快照模板 ID。
	SnapshotID string
	// Names 快照模板的名称，可能包含命名空间。
	Names []string
}

// ListSnapshotsParams 查询持久快照的过滤和分页参数。
type ListSnapshotsParams struct {
	// SandboxID 按来源沙箱过滤。
	SandboxID *string
	// Name 按名称或模板 ID 过滤，可包含命名空间和标签。
	Name *string
	// Limit 每页最大条数。
	Limit *int32
	// NextToken 上一页返回的分页游标。
	NextToken *string
}

// SnapshotPage 持久快照列表及下一页游标。
type SnapshotPage struct {
	// Items 当前页的快照。
	Items []SnapshotInfo
	// NextToken 下一页游标，空字符串表示没有下一页。
	NextToken string
}

// CreateSnapshot 创建指定沙箱的持久快照；同一次捕获的重试始终复用同一个幂等键和请求体。
// 每次新捕获都必须使用新键，包括同名快照的新版本。服务端在捕获完成后保留幂等记录 35 分钟；过期后重用该键可能发起新的捕获。
// 取消等待不会取消已接受的捕获。结果不确定时，使用 ListSnapshots 按来源沙箱或名称核对；核对前不要换键发起新捕获。
// 自动重试 408、502、504、error_code 为 api_shutdown 的 503 和网络错误；500 及其他 503 会直接返回。
func (c *Client) CreateSnapshot(ctx context.Context, sandboxID string, params CreateSnapshotParams) (*SnapshotInfo, error) {
	key := params.IdempotencyKey
	if key == "" || len(key) > 128 {
		return nil, fmt.Errorf("IdempotencyKey must contain between 1 and 128 bytes")
	}
	body := apis.CreateSandboxSnapshotJSONRequestBody{Memory: params.Memory, Name: params.Name}
	var result *SnapshotInfo
	err := c.retryCallWithPolicy(ctx, func() error {
		resp, err := c.api.CreateSandboxSnapshotWithResponse(ctx, sandboxID, &apis.CreateSandboxSnapshotParams{IdempotencyKey: key}, body)
		if err != nil {
			return err
		}
		if resp.JSON201 == nil {
			return newAPIError(resp.HTTPResponse, resp.Body)
		}
		result = &SnapshotInfo{SnapshotID: resp.JSON201.SnapshotID, Names: resp.JSON201.Names}
		return nil
	}, isSnapshotRetryable, snapshotRetryAfter)
	return result, err
}

func isSnapshotRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusRequestTimeout, http.StatusBadGateway, http.StatusGatewayTimeout:
			return true
		case http.StatusServiceUnavailable:
			return apiErr.Code == "api_shutdown"
		default:
			return false
		}
	}
	return isRetryableError(err)
}

func snapshotRetryAfter(err error) (time.Duration, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.retryAfter, apiErr.hasRetryAfter
	}
	return 0, false
}

// CreateSnapshot 创建当前沙箱的持久快照。
func (s *Sandbox) CreateSnapshot(ctx context.Context, params CreateSnapshotParams) (*SnapshotInfo, error) {
	return s.client.CreateSnapshot(ctx, s.sandboxID, params)
}

// ListSnapshots 分页查询团队持久快照。
func (c *Client) ListSnapshots(ctx context.Context, params *ListSnapshotsParams) (*SnapshotPage, error) {
	var query *apis.ListSnapshotsParams
	if params != nil {
		query = &apis.ListSnapshotsParams{SandboxID: params.SandboxID, Name: params.Name, Limit: params.Limit, NextToken: params.NextToken}
	}
	resp, err := c.api.ListSnapshotsWithResponse(ctx, query)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	page := &SnapshotPage{Items: make([]SnapshotInfo, len(*resp.JSON200)), NextToken: resp.HTTPResponse.Header.Get("X-Next-Token")}
	for i, item := range *resp.JSON200 {
		page.Items[i] = SnapshotInfo{SnapshotID: item.SnapshotID, Names: item.Names}
	}
	return page, nil
}
