package sandbox

import (
	"context"
)

// ListInjectionRules 列出当前用户的所有注入规则。
func (c *Client) ListInjectionRules(ctx context.Context) ([]InjectionRule, error) {
	resp, err := c.api.GetInjectionRulesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	return injectionRulesFromAPI(*resp.JSON200), nil
}

// CreateInjectionRule 创建一个新的注入规则。
func (c *Client) CreateInjectionRule(ctx context.Context, body CreateInjectionRuleParams) (*InjectionRule, error) {
	resp, err := c.api.PostInjectionRulesWithResponse(ctx, body.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON201 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r := injectionRuleFromAPI(*resp.JSON201)
	return &r, nil
}

// GetInjectionRule 获取指定注入规则的详情。
func (c *Client) GetInjectionRule(ctx context.Context, ruleID string) (*InjectionRule, error) {
	resp, err := c.api.GetInjectionRulesRuleIDWithResponse(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r := injectionRuleFromAPI(*resp.JSON200)
	return &r, nil
}

// UpdateInjectionRule 更新指定的注入规则。
func (c *Client) UpdateInjectionRule(ctx context.Context, ruleID string, body UpdateInjectionRuleParams) (*InjectionRule, error) {
	resp, err := c.api.PutInjectionRulesRuleIDWithResponse(ctx, ruleID, body.toAPI())
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, newAPIError(resp.HTTPResponse, resp.Body)
	}
	r := injectionRuleFromAPI(*resp.JSON200)
	return &r, nil
}

// DeleteInjectionRule 删除指定的注入规则。
func (c *Client) DeleteInjectionRule(ctx context.Context, ruleID string) error {
	resp, err := c.api.DeleteInjectionRulesRuleIDWithResponse(ctx, ruleID)
	if err != nil {
		return err
	}
	if resp.HTTPResponse.StatusCode != 204 {
		return newAPIError(resp.HTTPResponse, resp.Body)
	}
	return nil
}
