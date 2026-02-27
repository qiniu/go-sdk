package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/apis"
)

// ListTemplates 列出所有模板。
func (c *Client) ListTemplates(ctx context.Context, params *apis.ListTemplatesParams) ([]apis.Template, error) {
	resp, err := c.api.ListTemplatesWithResponse(ctx, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return *resp.JSON200, nil
}

// CreateTemplate 创建一个新模板（v3 API）。
func (c *Client) CreateTemplate(ctx context.Context, body apis.CreateTemplateV3JSONRequestBody) (*apis.TemplateRequestResponseV3, error) {
	resp, err := c.api.CreateTemplateV3WithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON202 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON202, nil
}

// GetTemplate 返回模板详情及其构建记录。
func (c *Client) GetTemplate(ctx context.Context, templateID string, params *apis.GetTemplateParams) (*apis.TemplateWithBuilds, error) {
	resp, err := c.api.GetTemplateWithResponse(ctx, templateID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}

// DeleteTemplate 删除一个模板。
func (c *Client) DeleteTemplate(ctx context.Context, templateID string) error {
	resp, err := c.api.DeleteTemplateWithResponse(ctx, templateID)
	if err != nil {
		return err
	}
	sc := resp.HTTPResponse.StatusCode
	if sc != 200 && sc != 204 {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// UpdateTemplate 更新一个模板。
func (c *Client) UpdateTemplate(ctx context.Context, templateID string, body apis.UpdateTemplateJSONRequestBody) error {
	resp, err := c.api.UpdateTemplateWithResponse(ctx, templateID, body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 200 {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// GetTemplateBuildStatus 返回模板的构建状态。
func (c *Client) GetTemplateBuildStatus(ctx context.Context, templateID, buildID string, params *apis.GetTemplateBuildStatusParams) (*apis.TemplateBuildInfo, error) {
	resp, err := c.api.GetTemplateBuildStatusWithResponse(ctx, templateID, buildID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}

// GetTemplateBuildLogs 返回模板的构建日志。
func (c *Client) GetTemplateBuildLogs(ctx context.Context, templateID, buildID string, params *apis.GetTemplateBuildLogsParams) (*apis.TemplateBuildLogsResponse, error) {
	resp, err := c.api.GetTemplateBuildLogsWithResponse(ctx, templateID, buildID, params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}

// StartTemplateBuild 启动模板构建（v2 API）。
func (c *Client) StartTemplateBuild(ctx context.Context, templateID, buildID string, body apis.StartTemplateBuildV2JSONRequestBody) error {
	resp, err := c.api.StartTemplateBuildV2WithResponse(ctx, templateID, buildID, body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 202 {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// GetTemplateFiles 返回模板构建文件的上传链接。
func (c *Client) GetTemplateFiles(ctx context.Context, templateID, hash string) (*apis.TemplateBuildFileUpload, error) {
	resp, err := c.api.GetTemplateFilesWithResponse(ctx, templateID, hash)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON201, nil
}

// GetTemplateByAlias 检查指定别名的模板是否存在。
func (c *Client) GetTemplateByAlias(ctx context.Context, alias string) (*apis.TemplateAliasResponse, error) {
	resp, err := c.api.GetTemplateByAliasWithResponse(ctx, alias)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON200, nil
}

// ManageTemplateTags 为模板构建分配标签。
func (c *Client) ManageTemplateTags(ctx context.Context, body apis.ManageTemplateTagsJSONRequestBody) (*apis.AssignedTemplateTags, error) {
	resp, err := c.api.ManageTemplateTagsWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return resp.JSON201, nil
}

// DeleteTemplateTags 删除模板的标签。
func (c *Client) DeleteTemplateTags(ctx context.Context, body apis.DeleteTemplateTagsJSONRequestBody) error {
	resp, err := c.api.DeleteTemplateTagsWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 204 {
		return &APIError{StatusCode: resp.StatusCode(), Body: resp.Body}
	}
	return nil
}

// WaitForBuild 轮询 GetTemplateBuildStatus 直到构建达到终态（"ready" 或 "error"）。
func (c *Client) WaitForBuild(ctx context.Context, templateID, buildID string, pollInterval time.Duration) (*apis.TemplateBuildInfo, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	for {
		info, err := c.GetTemplateBuildStatus(ctx, templateID, buildID, nil)
		if err != nil {
			return nil, fmt.Errorf("get build status %s/%s: %w", templateID, buildID, err)
		}
		switch info.Status {
		case apis.TemplateBuildStatusReady:
			return info, nil
		case apis.TemplateBuildStatusError:
			return info, fmt.Errorf("build %s/%s failed", templateID, buildID)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
