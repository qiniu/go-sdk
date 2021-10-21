package pili

import (
	"context"
	"net/http"
)

// GetDomainsList 查询域名列表
// GET /v2/hubs/<hub>/domains
func (m *Manager) GetDomainsList(ctx context.Context, req GetDomainsListRequest) (*GetDomainsListResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	response := new(GetDomainsListResponse)
	if err := m.client.Call(ctx, response, http.MethodGet, m.url("/v2/hubs/%s/domains", req.Hub), nil); err != nil {
		return nil, err
	}
	return response, nil
}

// GetDomainInfo 查询域名信息
// GET /v2/hubs/<hub>/domains/<domain>
func (m *Manager) GetDomainInfo(ctx context.Context, req GetDomainInfoRequest) (*GetDomainInfoResponse, error) {
	if err := defaultValidator.Validate(req); err != nil {
		return nil, err
	}

	response := new(GetDomainInfoResponse)
	if err := m.client.Call(ctx, response, http.MethodGet, m.url("/v2/hubs/%s/domains/%s", req.Hub, req.Domain), nil); err != nil {
		return nil, err
	}
	return response, nil
}

// BindDomain 绑定直播域名
// POST /v2/hubs/<hub>/newdomains
func (m *Manager) BindDomain(ctx context.Context, req BindDomainRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/newdomains", req.Hub), nil, req); err != nil {
		return err
	}
	return nil
}

// UnbindDomain 解绑直播域名
// DELETE /v2/hubs/<hub>/domains/<domain>
func (m *Manager) UnbindDomain(ctx context.Context, req UnbindDomainRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodDelete, m.url("/v2/hubs/%s/domains/%s", req.Hub, req.Domain), nil, req); err != nil {
		return err
	}
	return nil
}

// BindVodDomain 绑定点播域名
// POST /v2/hubs/<hub>/voddomain
// 点播域名用于访问直播空间对应的存储空间中的内容，例如回放、截图文件
// 请在存储空间控制台配置好可用域名后，绑定到直播空间
// 若未正确配置点播域名，可能无法正常使用回放录制、保存截图等功能
func (m *Manager) BindVodDomain(ctx context.Context, req BindVodDomainRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/voddomain", req.Hub), nil, req); err != nil {
		return err
	}
	return nil
}

// SetDomainCert 修改域名证书配置
// POST /v2/hubs/<hub>/domains/<domain>/cert
func (m *Manager) SetDomainCert(ctx context.Context, req SetDomainCertRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/domains/%s/cert", req.Hub, req.Domain), nil, req); err != nil {
		return err
	}
	return nil
}

// SetDomainURLRewrite 修改域名改写规则配置
// POST /v2/hubs/<hub>/domains/<domain>/urlrewrite
// 可根据业务需求自定义推拉流URL
// 改写后的URL应符合七牛的直播URL规范: <scheme>://<domain>/<hub>/<stream>[.<ext>]?<query>
// 举例
// 匹配规则: (.+)/live/(.+)/playlist.m3u8
// 改写规则: ${1}/hub/${2}.m3u8
// 请求URL: https://live.qiniu.com/live/stream01/playlist.m3u8 ; 改写URL: https://live.qiniu.com/hub/stream01.m3u8
// 请求URL: https://live.qiniu.com/live/stream01.m3u8 ; 与规则不匹配，不做改写
func (m *Manager) SetDomainURLRewrite(ctx context.Context, req SetDomainURLRewriteRequest) error {
	if err := defaultValidator.Validate(req); err != nil {
		return err
	}

	if err := m.client.CallWithJson(ctx, nil, http.MethodPost, m.url("/v2/hubs/%s/domains/%s/urlrewrite", req.Hub, req.Domain), nil, req); err != nil {
		return err
	}
	return nil
}
