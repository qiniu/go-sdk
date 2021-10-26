package pili

import (
	"context"
	"net/http"
)

// GetHubList 查询直播空间列表
// GET /v2/hubs
func (m *Manager) GetHubList(ctx context.Context) (*GetHubListResponse, error) {
	response := new(GetHubListResponse)
	if err := m.client.Call(ctx, response, http.MethodGet, m.apiHTTPScheme+m.apiHost+"/v2/hubs", nil); err != nil {
		return nil, err
	}
	return response, nil
}

// GetHubInfo 查询直播空间信息
// GET /v2/hubs/<hub>
func (m *Manager) GetHubInfo(ctx context.Context, req GetHubInfoRequest) (*GetHubInfoResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	response := new(GetHubInfoResponse)
	if err := m.client.Call(ctx, response, http.MethodGet, m.url("/v2/hubs/%s", req.Hub), nil); err != nil {
		return nil, err
	}
	response.Name = req.Hub
	return response, nil
}

// HubSecurity 修改直播空间推流鉴权配置
// POST /v2/hubs/<hub>/security
func (m *Manager) HubSecurity(ctx context.Context, req HubSecurityRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/security", req.Hub), nil, req); err != nil {
		return err
	}
	return nil
}

// HubHlsplus 修改直播空间 hls 低延迟配置
// POST /v2/hubs/<hub>/hlsplus
func (m *Manager) HubHlsplus(ctx context.Context, req HubHlsplusRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/hlsplus", req.Hub), nil, req); err != nil {
		return err
	}
	return nil
}

// HubPersistence 修改直播空间存储配置
// POST /v2/hubs/<hub>/persistence
func (m *Manager) HubPersistence(ctx context.Context, req HubPersistenceRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/persistence", req.Hub), nil, req); err != nil {
		return err
	}
	return nil
}

// HubSnapshot 修改直播空间封面配置
// POST /v2/hubs/<hub>/snapshot
func (m *Manager) HubSnapshot(ctx context.Context, req HubSnapshotRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/snapshot", req.Hub), nil, req); err != nil {
		return err
	}
	return nil
}
