package sandbox

import (
	"context"
)

// ListTransformRules 列出当前用户的所有转换规则。
func (c *Client) ListTransformRules(ctx context.Context) ([]TransformRule, error) {
	resp, err := c.api.GetTransformRulesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return transformRulesFromAPI(*resp.JSON200), nil
}

// CreateTransformRule 创建一个新的转换规则。
func (c *Client) CreateTransformRule(ctx context.Context, body CreateTransformRuleParams) (*TransformRule, error) {
	resp, err := c.api.PostTransformRulesWithResponse(ctx, body.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r := transformRuleFromAPI(*resp.JSON201)
	return &r, nil
}

// GetTransformRule 获取指定转换规则的详情。
func (c *Client) GetTransformRule(ctx context.Context, ruleID string) (*TransformRule, error) {
	resp, err := c.api.GetTransformRulesRuleIDWithResponse(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r := transformRuleFromAPI(*resp.JSON200)
	return &r, nil
}

// UpdateTransformRule 更新指定的转换规则。
func (c *Client) UpdateTransformRule(ctx context.Context, ruleID string, body UpdateTransformRuleParams) (*TransformRule, error) {
	resp, err := c.api.PutTransformRulesRuleIDWithResponse(ctx, ruleID, body.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r := transformRuleFromAPI(*resp.JSON200)
	return &r, nil
}

// DeleteTransformRule 删除指定的转换规则。
func (c *Client) DeleteTransformRule(ctx context.Context, ruleID string) error {
	resp, err := c.api.DeleteTransformRulesRuleIDWithResponse(ctx, ruleID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 204 {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}
