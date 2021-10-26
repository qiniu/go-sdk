package pili

import (
	"context"
	"net/http"
)

// GetStatUpflow 获取上行流量
// GET /statd/upflow
func (m *Manager) GetStatUpflow(ctx context.Context, req GetStatUpflowRequest) ([]StatResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{FlowDefaultSelect}
	}

	var resp []StatResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/upflow"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GroupStatUpflow 分组获取上行流量
// GET /statd/upflow
func (m *Manager) GroupStatUpflow(ctx context.Context, req GroupStatUpflowRequest) ([]StatGroupResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{FlowDefaultSelect}
	}

	var resp []StatGroupResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/upflow"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStatDownflow 获取下行流量
// GET /statd/downflow
func (m *Manager) GetStatDownflow(ctx context.Context, req GetStatDownflowRequest) ([]StatResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{FlowDefaultSelect}
	}

	var resp []StatResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/downflow"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GroupStatDownflow 分组获取下行流量
// GET /statd/downflow
func (m *Manager) GroupStatDownflow(ctx context.Context, req GroupStatDownflowRequest) ([]StatGroupResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{FlowDefaultSelect}
	}

	var resp []StatGroupResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/downflow"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStatCodec 获取直播转码使用量
// GET /statd/codec
func (m *Manager) GetStatCodec(ctx context.Context, req GetStatCodecRequest) ([]StatResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{CodecDefaultSelect}
	}

	var resp []StatResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/codec"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GroupStatCodec 分组获取直播转码使用量
// GET /statd/codec
func (m *Manager) GroupStatCodec(ctx context.Context, req GroupStatCodecRequest) ([]StatGroupResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{CodecDefaultSelect}
	}

	var resp []StatGroupResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/codec"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStatNrop 获取直播鉴黄使用量
// GET /statd/nrop
func (m *Manager) GetStatNrop(ctx context.Context, req GetStatNropRequest) ([]StatResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{NropDefaultSelect}
	}

	var resp []StatResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/nrop"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GroupStatNrop 分组获取直播鉴黄使用量
// GET /statd/nrop
func (m *Manager) GroupStatNrop(ctx context.Context, req GroupStatNropRequest) ([]StatGroupResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}
	if len(req.Select) == 0 {
		req.Select = []string{NropDefaultSelect}
	}

	var resp []StatGroupResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/nrop"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStatCaster 获取导播台使用量
// GET /statd/caster
func (m *Manager) GetStatCaster(ctx context.Context, req GetStatCasterRequest) ([]StatResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	var resp []StatResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/caster"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GroupStatCaster 分组获取导播台使用量
// GET /statd/caster
func (m *Manager) GroupStatCaster(ctx context.Context, req GroupStatCasterRequest) ([]StatGroupResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	var resp []StatGroupResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/caster"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStatPub 获取Pub服务使用量
// GET /statd/pub
func (m *Manager) GetStatPub(ctx context.Context, req GetStatPubRequest) ([]StatResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	var resp []StatResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/pub"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}

// GroupStatPub 分组获取Pub服务使用量
// GET /statd/pub
func (m *Manager) GroupStatPub(ctx context.Context, req GroupStatPubRequest) ([]StatGroupResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	var resp []StatGroupResponse
	if err := m.client.CallWithForm(ctx, &resp, http.MethodGet, m.url("/statd/pub"), nil, Form(req)); err != nil {
		return nil, err
	}
	return resp, nil
}
