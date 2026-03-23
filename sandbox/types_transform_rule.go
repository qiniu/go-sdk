package sandbox

import (
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK 自有类型 — 请求转换规则相关
// ---------------------------------------------------------------------------

// TransformRule 预定义的请求转换规则。
type TransformRule struct {
	// RuleID 规则唯一标识。
	RuleID string

	// Name 规则名称，同一用户下唯一。
	Name string

	// Conditions 匹配条件。
	Conditions *RequestTransformConditions

	// Replacements 替换动作。
	Replacements *RequestTransformReplacements

	// CreatedAt 创建时间。
	CreatedAt time.Time

	// UpdatedAt 最后更新时间。
	UpdatedAt time.Time
}

// CreateTransformRuleParams 创建转换规则的请求参数。
type CreateTransformRuleParams struct {
	// Name 规则名称（必填），同一用户下唯一。
	Name string

	// Conditions 匹配条件，可选。
	Conditions *RequestTransformConditions

	// Replacements 替换动作，可选。
	Replacements *RequestTransformReplacements
}

func (p *CreateTransformRuleParams) toAPI() apis.PostTransformRulesJSONRequestBody {
	body := apis.PostTransformRulesJSONRequestBody{
		Name: p.Name,
	}
	if p.Conditions != nil {
		body.Conditions = &apis.RequestTransformConditions{
			Hosts: p.Conditions.Hosts,
		}
	}
	if p.Replacements != nil {
		body.Replacements = &apis.RequestTransformReplacements{
			Headers: p.Replacements.Headers,
			Queries: p.Replacements.Queries,
		}
	}
	return body
}

// UpdateTransformRuleParams 更新转换规则的请求参数。
type UpdateTransformRuleParams struct {
	// Name 规则名称，可选。
	Name *string

	// Conditions 匹配条件，可选。
	Conditions *RequestTransformConditions

	// Replacements 替换动作，可选。
	Replacements *RequestTransformReplacements
}

func (p *UpdateTransformRuleParams) toAPI() apis.PutTransformRulesRuleIDJSONRequestBody {
	body := apis.PutTransformRulesRuleIDJSONRequestBody{
		Name: p.Name,
	}
	if p.Conditions != nil {
		body.Conditions = &apis.RequestTransformConditions{
			Hosts: p.Conditions.Hosts,
		}
	}
	if p.Replacements != nil {
		body.Replacements = &apis.RequestTransformReplacements{
			Headers: p.Replacements.Headers,
			Queries: p.Replacements.Queries,
		}
	}
	return body
}

// ---------------------------------------------------------------------------
// 转换函数 — apis → SDK
// ---------------------------------------------------------------------------

func transformRuleFromAPI(a apis.TransformRule) TransformRule {
	r := TransformRule{
		RuleID:    a.RuleID,
		Name:      a.Name,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	if a.Conditions != nil {
		r.Conditions = &RequestTransformConditions{
			Hosts: a.Conditions.Hosts,
		}
	}
	if a.Replacements != nil {
		r.Replacements = &RequestTransformReplacements{
			Headers: a.Replacements.Headers,
			Queries: a.Replacements.Queries,
		}
	}
	return r
}

func transformRulesFromAPI(a []apis.TransformRule) []TransformRule {
	if a == nil {
		return nil
	}
	result := make([]TransformRule, len(a))
	for i, r := range a {
		result[i] = transformRuleFromAPI(r)
	}
	return result
}
