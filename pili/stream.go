package pili

import (
	"context"
	"net/http"
	"net/url"
)

// GetStreamsList 查询直播流列表
// GET /v2/hubs/<Hub>/streams?liveonly=<true>&prefix=<Prefix>&limit=<Limit>&marker=<Marker>
func (m *Manager) GetStreamsList(ctx context.Context, req GetStreamListRequest) (*GetStreamsListResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	query := url.Values{}
	setQuery(query, "liveonly", req.LiveOnly)
	setQuery(query, "prefix", req.Prefix)
	setQuery(query, "limit", req.Limit)
	setQuery(query, "marker", req.Marker)
	resp := GetStreamsListResponse{}
	if err := m.client.Call(ctx, &resp, http.MethodGet, m.url("/v2/hubs/%s/streams", req.Hub, query), nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetStreamBaseInfo 查询直播流信息
// GET v2/hubs/<hub>/streams/<EncodedStreamTitle>
func (m *Manager) GetStreamBaseInfo(ctx context.Context, req GetStreamBaseInfoRequest) (*GetStreamBaseInfoResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	resp := GetStreamBaseInfoResponse{}
	if err := m.client.Call(ctx, &resp, http.MethodGet, m.url("/v2/hubs/%s/streams/%s", req.Hub, encodeStream(req.Stream)), nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StreamDisable 禁用直播流
// POST /v2/hubs/<hub>/streams/<EncodedStreamTitle>/disabled
func (m *Manager) StreamDisable(ctx context.Context, req StreamDisabledRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/streams/%s/disabled", req.Hub, encodeStream(req.Stream)), nil, req); err != nil {
		return err
	}
	return nil
}

// GetStreamLiveStatus 查询直播流实时信息
// GET v2/hubs/<hub>/streams/<EncodedStreamTitle>/live
func (m *Manager) GetStreamLiveStatus(ctx context.Context, req GetStreamLiveStatusRequest) (*GetStreamLiveStatusResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	resp := GetStreamLiveStatusResponse{}
	if err := m.client.Call(ctx, &resp, http.MethodGet, m.url("/v2/hubs/%s/streams/%s/live", req.Hub, encodeStream(req.Stream)), nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchGetStreamLiveStatus 批量查询直播实时状态
// POST /v2/hubs/<hub>/livestreams
func (m *Manager) BatchGetStreamLiveStatus(ctx context.Context, req BatchGetStreamLiveStatusRequest) (*BatchGetStreamLiveStatusResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	resp := BatchGetStreamLiveStatusResponse{}
	if err := m.client.CallWithJson(ctx, &resp, http.MethodPost, m.url("/v2/hubs/%s/livestreams", req.Hub), nil, req); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetStreamHistory 查询直播流推流记录
// GET /v2/hubs/<hub>/streams/<EncodedStreamTitle>/historyactivity?start=<Start>&end=<End>
func (m *Manager) GetStreamHistory(ctx context.Context, req GetStreamHistoryRequest) (*GetStreamHistoryResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	query := url.Values{}
	setQuery(query, "start", req.Start)
	setQuery(query, "end", req.End)
	resp := GetStreamHistoryResponse{}
	if err := m.client.Call(ctx, &resp, http.MethodGet, m.url("/v2/hubs/%s/streams/%s/historyactivity", req.Hub, encodeStream(req.Stream), query), nil); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StreamSaveas 录制直播回放
// POST /v2/hubs/<hub>/streams/<EncodedStreamTitle>/saveas
func (m *Manager) StreamSaveas(ctx context.Context, req StreamSaveasRequest) (*StreamSaveasResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	resp := StreamSaveasResponse{}
	if err := m.client.CallWithJson(ctx, &resp, http.MethodPost, m.url("/v2/hubs/%s/streams/%s/saveas", req.Hub, encodeStream(req.Stream)), nil, req); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StreamSnapshot 保存直播截图
// POST /v2/hubs/<hub>/streams/<EncodedStreamTitle>/snapshot
func (m *Manager) StreamSnapshot(ctx context.Context, req StreamSnapshotRequest) (*StreamSnapshotResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	resp := StreamSnapshotResponse{}
	if err := m.client.CallWithJson(ctx, &resp, http.MethodPost, m.url("/v2/hubs/%s/streams/%s/snapshot", req.Hub, encodeStream(req.Stream)), nil, req); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StreamConverts 修改直播流转码配置
// POST /v2/hubs/<hub>/streams/<EncodedStreamTitle>/converts
func (m *Manager) StreamConverts(ctx context.Context, req StreamConvertsRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/streams/%s/converts", req.Hub, encodeStream(req.Stream)), nil, req); err != nil {
		return err
	}
	return nil
}
