package sandbox

import (
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK 自有类型 — 请求注入规则相关
// ---------------------------------------------------------------------------

// InjectionRule 预定义的请求注入规则。
type InjectionRule struct {
	// RuleID 规则唯一标识。
	RuleID string

	// Name 规则名称，同一用户下唯一。
	Name string

	// Conditions 匹配条件。
	Conditions *RequestInjectionConditions

	// Injections 注入动作。
	Injections *RequestInjections

	// CreatedAt 创建时间。
	CreatedAt time.Time

	// UpdatedAt 最后更新时间。
	UpdatedAt time.Time
}

// CreateInjectionRuleParams 创建注入规则的请求参数。
type CreateInjectionRuleParams struct {
	// Name 规则名称（必填），同一用户下唯一。
	Name string

	// Conditions 匹配条件，可选。
	Conditions *RequestInjectionConditions

	// Injections 注入动作，可选。
	Injections *RequestInjections
}

func (p *CreateInjectionRuleParams) toAPI() apis.PostInjectionRulesJSONRequestBody {
	body := apis.PostInjectionRulesJSONRequestBody{
		Name: p.Name,
	}
	if p.Conditions != nil {
		body.Conditions = &apis.RequestInjectionConditions{
			Hosts: p.Conditions.Hosts,
		}
	}
	if p.Injections != nil {
		body.Injections = injectionsToAPI(p.Injections)
	}
	return body
}

// UpdateInjectionRuleParams 更新注入规则的请求参数。
type UpdateInjectionRuleParams struct {
	// Name 规则名称，可选。
	Name *string

	// Conditions 匹配条件，可选。
	Conditions *RequestInjectionConditions

	// Injections 注入动作，可选。
	Injections *RequestInjections
}

func (p *UpdateInjectionRuleParams) toAPI() apis.PutInjectionRulesRuleIDJSONRequestBody {
	body := apis.PutInjectionRulesRuleIDJSONRequestBody{
		Name: p.Name,
	}
	if p.Conditions != nil {
		body.Conditions = &apis.RequestInjectionConditions{
			Hosts: p.Conditions.Hosts,
		}
	}
	if p.Injections != nil {
		body.Injections = injectionsToAPI(p.Injections)
	}
	return body
}

// ---------------------------------------------------------------------------
// 转换函数 — apis → SDK / SDK → apis
// ---------------------------------------------------------------------------

func injectionsToAPI(inj *RequestInjections) *apis.RequestInjections {
	result := &apis.RequestInjections{
		Headers: inj.Headers,
		Queries: inj.Queries,
	}
	if inj.API != nil {
		result.API = &apis.APIKeyInjection{
			Type:  inj.API.Type,
			Value: inj.API.Value,
		}
	}
	return result
}

func injectionRuleFromAPI(a apis.InjectionRule) InjectionRule {
	r := InjectionRule{
		RuleID:    a.RuleID,
		Name:      a.Name,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	if a.Conditions != nil {
		r.Conditions = &RequestInjectionConditions{
			Hosts: a.Conditions.Hosts,
		}
	}
	if a.Injections != nil {
		r.Injections = &RequestInjections{
			Headers: a.Injections.Headers,
			Queries: a.Injections.Queries,
		}
		if a.Injections.API != nil {
			r.Injections.API = &APIKeyInjection{
				Type:  a.Injections.API.Type,
				Value: a.Injections.API.Value,
			}
		}
	}
	return r
}

func injectionRulesFromAPI(a []apis.InjectionRule) []InjectionRule {
	if a == nil {
		return nil
	}
	result := make([]InjectionRule, len(a))
	for i, r := range a {
		result[i] = injectionRuleFromAPI(r)
	}
	return result
}
