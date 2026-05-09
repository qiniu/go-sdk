package sandbox

import (
	"fmt"
	"time"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

// ---------------------------------------------------------------------------
// SDK 自有类型 — 注入配置（含注入规则）
// ---------------------------------------------------------------------------

// OpenAIInjection OpenAI 兼容 API 注入配置。自动设置 Authorization: Bearer 头。
// 默认 host: api.openai.com
type OpenAIInjection struct {
	// APIKey API 密钥，可选。
	APIKey *string

	// BaseURL 可选 base URL，未指定时使用 api.openai.com。
	BaseURL *string
}

// AnthropicInjection Anthropic API 注入配置。自动设置 x-api-key 头。
// 默认 host: api.anthropic.com
type AnthropicInjection struct {
	// APIKey API 密钥，可选。
	APIKey *string

	// BaseURL 可选 base URL，未指定时使用 api.anthropic.com。
	BaseURL *string
}

// GeminiInjection Google Gemini API 注入配置。自动设置 x-goog-api-key 头。
// 默认 host: generativelanguage.googleapis.com
type GeminiInjection struct {
	// APIKey API 密钥，可选。
	APIKey *string

	// BaseURL 可选 base URL，未指定时使用 generativelanguage.googleapis.com。
	BaseURL *string
}

// QiniuInjection 七牛 AI API 注入配置。七牛网关同时兼容 OpenAI 和 Anthropic 协议，
// api_key 会注入到请求实际携带的认证头（Authorization 或 x-api-key）。
// 默认 host: api.qnaigc.com
type QiniuInjection struct {
	// APIKey API 密钥。
	APIKey *string

	// BaseURL 可选 base URL，未指定时使用 api.qnaigc.com。
	BaseURL *string
}

// GithubInjection GitHub 凭证注入。平台用此凭证克隆并校验仓库，且在沙箱运行期间为
// 匹配 github.com / api.github.com 的 HTTPS 请求自动注入认证；token 不会以明文形式
// 暴露到沙箱内。
type GithubInjection struct {
	// Token GitHub token，对所有请求的仓库具备访问权限。
	Token *string
}

// HTTPInjection 自定义 HTTP 注入配置。
type HTTPInjection struct {
	// BaseURL 匹配 HTTPS 请求的 base URL。域名部分用于 host 匹配。
	// 若未指定 scheme 默认为 https。
	BaseURL string

	// Headers 需要注入或覆盖的 HTTP Headers，可选。
	Headers *map[string]string
}

// InjectionSpec 注入配置（discriminated union），各字段互斥，只能设置一个。
type InjectionSpec struct {
	// OpenAI OpenAI 兼容 API 注入。
	OpenAI *OpenAIInjection

	// Anthropic Anthropic API 注入。
	Anthropic *AnthropicInjection

	// Gemini Google Gemini API 注入。
	Gemini *GeminiInjection

	// Qiniu 七牛 AI API 注入。
	Qiniu *QiniuInjection

	// Github GitHub 凭证注入。
	Github *GithubInjection

	// HTTP 自定义 HTTP 注入。
	HTTP *HTTPInjection
}

// SandboxInjectionSpec 沙箱注入配置（discriminated union）。
// 可引用已保存的注入规则 ID，或直接指定注入配置，各字段互斥。
type SandboxInjectionSpec struct {
	// ByID 引用已保存的注入规则 ID。
	ByID *string

	// OpenAI OpenAI 兼容 API 注入。
	OpenAI *OpenAIInjection

	// Anthropic Anthropic API 注入。
	Anthropic *AnthropicInjection

	// Gemini Google Gemini API 注入。
	Gemini *GeminiInjection

	// Qiniu 七牛 AI API 注入。
	Qiniu *QiniuInjection

	// Github GitHub 凭证注入。
	Github *GithubInjection

	// HTTP 自定义 HTTP 注入。
	HTTP *HTTPInjection
}

// InjectionRule 预定义的请求注入规则。
type InjectionRule struct {
	// RuleID 规则唯一标识。
	RuleID string

	// Name 规则名称，同一用户下唯一。
	Name string

	// Injection 注入配置。
	Injection InjectionSpec

	// CreatedAt 创建时间。
	CreatedAt time.Time

	// UpdatedAt 最后更新时间。
	UpdatedAt time.Time
}

// CreateInjectionRuleParams 创建注入规则的请求参数。
type CreateInjectionRuleParams struct {
	// Name 规则名称（必填），同一用户下唯一。
	Name string

	// Injection 注入配置（必填）。
	Injection InjectionSpec
}

func (p *CreateInjectionRuleParams) toAPI() (apis.PostInjectionRulesJSONRequestBody, error) {
	inj, err := injectionSpecToAPI(p.Injection)
	if err != nil {
		return apis.PostInjectionRulesJSONRequestBody{}, err
	}
	return apis.PostInjectionRulesJSONRequestBody{
		Name:      p.Name,
		Injection: inj,
	}, nil
}

// UpdateInjectionRuleParams 更新注入规则的请求参数。
type UpdateInjectionRuleParams struct {
	// Name 规则名称，可选。
	Name *string

	// Injection 注入配置，可选。
	Injection *InjectionSpec
}

func (p *UpdateInjectionRuleParams) toAPI() (apis.PutInjectionRulesRuleIDJSONRequestBody, error) {
	body := apis.PutInjectionRulesRuleIDJSONRequestBody{
		Name: p.Name,
	}
	if p.Injection != nil {
		inj, err := injectionSpecToAPI(*p.Injection)
		if err != nil {
			return body, err
		}
		body.Injection = &inj
	}
	return body, nil
}

// ---------------------------------------------------------------------------
// 转换函数 — SDK → apis
// ---------------------------------------------------------------------------

func injectionSpecToAPI(spec InjectionSpec) (apis.Injection, error) {
	count := 0
	if spec.OpenAI != nil {
		count++
	}
	if spec.Anthropic != nil {
		count++
	}
	if spec.Gemini != nil {
		count++
	}
	if spec.HTTP != nil {
		count++
	}
	if spec.Qiniu != nil {
		count++
	}
	if spec.Github != nil {
		count++
	}
	if count == 0 {
		return apis.Injection{}, fmt.Errorf("InjectionSpec: exactly one injection type must be set (OpenAI, Anthropic, Gemini, Qiniu, GitHub, or HTTP), got none")
	}
	if count > 1 {
		return apis.Injection{}, fmt.Errorf("InjectionSpec: exactly one injection type must be set, but got %d", count)
	}

	var inj apis.Injection
	var err error
	switch {
	case spec.OpenAI != nil:
		err = inj.FromOpenaiInjection(apis.OpenaiInjection{
			APIKey:  spec.OpenAI.APIKey,
			BaseURL: spec.OpenAI.BaseURL,
			Type:    apis.Openai,
		})
	case spec.Anthropic != nil:
		err = inj.FromAnthropicInjection(apis.AnthropicInjection{
			APIKey:  spec.Anthropic.APIKey,
			BaseURL: spec.Anthropic.BaseURL,
			Type:    apis.Anthropic,
		})
	case spec.Gemini != nil:
		err = inj.FromGeminiInjection(apis.GeminiInjection{
			APIKey:  spec.Gemini.APIKey,
			BaseURL: spec.Gemini.BaseURL,
			Type:    apis.Gemini,
		})
	case spec.Qiniu != nil:
		err = inj.FromQiniuInjection(apis.QiniuInjection{
			APIKey:  spec.Qiniu.APIKey,
			BaseURL: spec.Qiniu.BaseURL,
			Type:    apis.Qiniu,
		})
	case spec.Github != nil:
		err = inj.FromGithubInjection(apis.GithubInjection{
			Token: spec.Github.Token,
			Type:  apis.Github,
		})
	case spec.HTTP != nil:
		err = inj.FromHTTPInjection(apis.HTTPInjection{
			BaseURL: spec.HTTP.BaseURL,
			Headers: spec.HTTP.Headers,
			Type:    apis.HTTP,
		})
	}
	return inj, err
}

func sandboxInjectionSpecToAPI(spec SandboxInjectionSpec) (apis.SandboxInjection, error) {
	count := 0
	if spec.ByID != nil {
		count++
	}
	if spec.OpenAI != nil {
		count++
	}
	if spec.Anthropic != nil {
		count++
	}
	if spec.Gemini != nil {
		count++
	}
	if spec.Qiniu != nil {
		count++
	}
	if spec.Github != nil {
		count++
	}
	if spec.HTTP != nil {
		count++
	}
	if count == 0 {
		return apis.SandboxInjection{}, fmt.Errorf("SandboxInjectionSpec: exactly one injection type must be set (ByID, OpenAI, Anthropic, Gemini, Qiniu, GitHub, or HTTP), got none")
	}
	if count > 1 {
		return apis.SandboxInjection{}, fmt.Errorf("SandboxInjectionSpec: exactly one injection type must be set, but got %d", count)
	}

	var si apis.SandboxInjection
	var err error
	switch {
	case spec.ByID != nil:
		err = si.FromInjectionByID(apis.InjectionByID{
			ID:   *spec.ByID,
			Type: apis.ID,
		})
	case spec.OpenAI != nil:
		err = si.FromOpenaiInjection(apis.OpenaiInjection{
			APIKey:  spec.OpenAI.APIKey,
			BaseURL: spec.OpenAI.BaseURL,
			Type:    apis.Openai,
		})
	case spec.Anthropic != nil:
		err = si.FromAnthropicInjection(apis.AnthropicInjection{
			APIKey:  spec.Anthropic.APIKey,
			BaseURL: spec.Anthropic.BaseURL,
			Type:    apis.Anthropic,
		})
	case spec.Gemini != nil:
		err = si.FromGeminiInjection(apis.GeminiInjection{
			APIKey:  spec.Gemini.APIKey,
			BaseURL: spec.Gemini.BaseURL,
			Type:    apis.Gemini,
		})
	case spec.Qiniu != nil:
		err = si.FromQiniuInjection(apis.QiniuInjection{
			APIKey:  spec.Qiniu.APIKey,
			BaseURL: spec.Qiniu.BaseURL,
			Type:    apis.Qiniu,
		})
	case spec.Github != nil:
		err = si.FromGithubInjection(apis.GithubInjection{
			Token: spec.Github.Token,
			Type:  apis.Github,
		})
	case spec.HTTP != nil:
		err = si.FromHTTPInjection(apis.HTTPInjection{
			BaseURL: spec.HTTP.BaseURL,
			Headers: spec.HTTP.Headers,
			Type:    apis.HTTP,
		})
	}
	return si, err
}

// ---------------------------------------------------------------------------
// 转换函数 — apis → SDK
// ---------------------------------------------------------------------------

func injectionSpecFromAPI(inj apis.Injection) (InjectionSpec, error) {
	disc, err := inj.Discriminator()
	if err != nil {
		return InjectionSpec{}, err
	}
	switch disc {
	case string(apis.Openai):
		v, err := inj.AsOpenaiInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{OpenAI: &OpenAIInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.Anthropic):
		v, err := inj.AsAnthropicInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{Anthropic: &AnthropicInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.Gemini):
		v, err := inj.AsGeminiInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{Gemini: &GeminiInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.Qiniu):
		v, err := inj.AsQiniuInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{Qiniu: &QiniuInjection{APIKey: v.APIKey, BaseURL: v.BaseURL}}, nil
	case string(apis.Github):
		v, err := inj.AsGithubInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{Github: &GithubInjection{Token: v.Token}}, nil
	case string(apis.HTTP):
		v, err := inj.AsHTTPInjection()
		if err != nil {
			return InjectionSpec{}, err
		}
		return InjectionSpec{HTTP: &HTTPInjection{BaseURL: v.BaseURL, Headers: v.Headers}}, nil
	default:
		return InjectionSpec{}, fmt.Errorf("unknown injection type: %s", disc)
	}
}

func injectionRuleFromAPI(a apis.InjectionRule) (InjectionRule, error) {
	spec, err := injectionSpecFromAPI(a.Injection)
	if err != nil {
		return InjectionRule{}, err
	}
	return InjectionRule{
		RuleID:    a.RuleID,
		Name:      a.Name,
		Injection: spec,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}, nil
}

func injectionRulesFromAPI(a []apis.InjectionRule) ([]InjectionRule, error) {
	if a == nil {
		return nil, nil
	}
	result := make([]InjectionRule, len(a))
	for i, r := range a {
		rule, err := injectionRuleFromAPI(r)
		if err != nil {
			return nil, err
		}
		result[i] = rule
	}
	return result, nil
}
