//go:build unit

package sandbox

import (
	"reflect"
	"strings"
	"testing"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

func TestSandboxInjectionSpecToAPISupportsAllTypes(t *testing.T) {
	value := "secret"
	headers := map[string]string{"Authorization": "Bearer secret"}
	tests := []struct {
		name string
		spec SandboxInjectionSpec
		want string
	}{
		{name: "by ID", spec: SandboxInjectionSpec{ByID: &value}, want: string(apis.ID)},
		{name: "OpenAI", spec: SandboxInjectionSpec{OpenAI: &OpenAIInjection{APIKey: &value}}, want: string(apis.Openai)},
		{name: "Anthropic", spec: SandboxInjectionSpec{Anthropic: &AnthropicInjection{APIKey: &value}}, want: string(apis.Anthropic)},
		{name: "Gemini", spec: SandboxInjectionSpec{Gemini: &GeminiInjection{APIKey: &value}}, want: string(apis.Gemini)},
		{name: "Qiniu", spec: SandboxInjectionSpec{Qiniu: &QiniuInjection{APIKey: &value}}, want: string(apis.Qiniu)},
		{name: "GitHub", spec: SandboxInjectionSpec{Github: &GithubInjection{Token: &value}}, want: string(apis.Github)},
		{name: "HTTP", spec: SandboxInjectionSpec{HTTP: &HTTPInjection{BaseURL: "https://example.com", Headers: &headers}}, want: string(apis.HTTP)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sandboxInjectionSpecToAPI(tt.spec)
			if err != nil {
				t.Fatalf("sandboxInjectionSpecToAPI() error = %v", err)
			}
			discriminator, err := got.Discriminator()
			if err != nil {
				t.Fatalf("Discriminator() error = %v", err)
			}
			if discriminator != tt.want {
				t.Fatalf("Discriminator() = %q, want %q", discriminator, tt.want)
			}
		})
	}
}

func TestSandboxInjectionSpecToAPIRequiresExactlyOneType(t *testing.T) {
	value := "secret"
	tests := []struct {
		name string
		spec SandboxInjectionSpec
		want string
	}{
		{name: "none", spec: SandboxInjectionSpec{}, want: "got none"},
		{name: "multiple", spec: SandboxInjectionSpec{ByID: &value, Github: &GithubInjection{Token: &value}}, want: "got 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sandboxInjectionSpecToAPI(tt.spec)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("sandboxInjectionSpecToAPI() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestMaskedSandboxInjectionFromAPISupportsAllTypes(t *testing.T) {
	value := "masked"
	headers := map[string]string{"Authorization": "Bear****"}
	tests := []struct {
		name  string
		build func(*apis.SandboxInjection) error
		check func(MaskedSandboxInjection) bool
	}{
		{name: "by ID", build: func(v *apis.SandboxInjection) error {
			return v.FromInjectionByID(apis.InjectionByID{ID: value, Type: apis.ID})
		}, check: func(v MaskedSandboxInjection) bool { return v.ByID != nil }},
		{name: "OpenAI", build: func(v *apis.SandboxInjection) error {
			return v.FromOpenaiInjection(apis.OpenaiInjection{APIKey: &value, Type: apis.Openai})
		}, check: func(v MaskedSandboxInjection) bool { return v.OpenAI != nil }},
		{name: "Anthropic", build: func(v *apis.SandboxInjection) error {
			return v.FromAnthropicInjection(apis.AnthropicInjection{APIKey: &value, Type: apis.Anthropic})
		}, check: func(v MaskedSandboxInjection) bool { return v.Anthropic != nil }},
		{name: "Gemini", build: func(v *apis.SandboxInjection) error {
			return v.FromGeminiInjection(apis.GeminiInjection{APIKey: &value, Type: apis.Gemini})
		}, check: func(v MaskedSandboxInjection) bool { return v.Gemini != nil }},
		{name: "Qiniu", build: func(v *apis.SandboxInjection) error {
			return v.FromQiniuInjection(apis.QiniuInjection{APIKey: &value, Type: apis.Qiniu})
		}, check: func(v MaskedSandboxInjection) bool { return v.Qiniu != nil }},
		{name: "GitHub", build: func(v *apis.SandboxInjection) error {
			return v.FromGithubInjection(apis.GithubInjection{Token: &value, Type: apis.Github})
		}, check: func(v MaskedSandboxInjection) bool { return v.Github != nil }},
		{name: "HTTP", build: func(v *apis.SandboxInjection) error {
			return v.FromHTTPInjection(apis.HTTPInjection{BaseURL: "https://example.com", Headers: &headers, Type: apis.HTTP})
		}, check: func(v MaskedSandboxInjection) bool { return v.HTTP != nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiInjection apis.SandboxInjection
			if err := tt.build(&apiInjection); err != nil {
				t.Fatalf("build API injection: %v", err)
			}
			got, err := maskedSandboxInjectionFromAPI(apiInjection)
			if err != nil {
				t.Fatalf("maskedSandboxInjectionFromAPI() error = %v", err)
			}
			if !tt.check(got) {
				t.Fatalf("maskedSandboxInjectionFromAPI() returned wrong variant: %+v", got)
			}
		})
	}
}

func TestInjectionSpecToAPIPreservesMatchConditions(t *testing.T) {
	apiKey := "sk-test"
	baseURL := "https://api.openai.com/v1/chat/*"
	ifHeaders := map[string]string{"X-Use-Model": "gpt"}
	ifQueries := map[string]string{"tenant": "demo"}

	inj, err := injectionSpecToAPI(InjectionSpec{
		OpenAI: &OpenAIInjection{
			APIKey:    &apiKey,
			BaseURL:   &baseURL,
			IfHeaders: &ifHeaders,
			IfQueries: &ifQueries,
		},
	})
	if err != nil {
		t.Fatalf("injectionSpecToAPI() error = %v", err)
	}

	got, err := inj.AsOpenaiInjection()
	if err != nil {
		t.Fatalf("AsOpenaiInjection() error = %v", err)
	}
	if got.APIKey == nil || *got.APIKey != apiKey {
		t.Fatalf("APIKey = %v, want %q", got.APIKey, apiKey)
	}
	if got.BaseURL == nil || *got.BaseURL != baseURL {
		t.Fatalf("BaseURL = %v, want %q", got.BaseURL, baseURL)
	}
	if !reflect.DeepEqual(got.IfHeaders, &ifHeaders) {
		t.Fatalf("IfHeaders = %#v, want %#v", got.IfHeaders, &ifHeaders)
	}
	if !reflect.DeepEqual(got.IfQueries, &ifQueries) {
		t.Fatalf("IfQueries = %#v, want %#v", got.IfQueries, &ifQueries)
	}
}

func TestSandboxInjectionSpecToAPIPreservesGithubMatchConditions(t *testing.T) {
	token := "ghp-test"
	baseURL := "https://api.github.com/repos/qiniu/*"
	ifHeaders := map[string]string{"X-GitHub-Api-Version": "2022-11-28"}
	ifQueries := map[string]string{"per_page": "100"}

	inj, err := sandboxInjectionSpecToAPI(SandboxInjectionSpec{
		Github: &GithubInjection{
			BaseURL:   &baseURL,
			IfHeaders: &ifHeaders,
			IfQueries: &ifQueries,
			Token:     &token,
		},
	})
	if err != nil {
		t.Fatalf("sandboxInjectionSpecToAPI() error = %v", err)
	}

	got, err := inj.AsGithubInjection()
	if err != nil {
		t.Fatalf("AsGithubInjection() error = %v", err)
	}
	if got.BaseURL == nil || *got.BaseURL != baseURL {
		t.Fatalf("BaseURL = %v, want %q", got.BaseURL, baseURL)
	}
	if !reflect.DeepEqual(got.IfHeaders, &ifHeaders) {
		t.Fatalf("IfHeaders = %#v, want %#v", got.IfHeaders, &ifHeaders)
	}
	if !reflect.DeepEqual(got.IfQueries, &ifQueries) {
		t.Fatalf("IfQueries = %#v, want %#v", got.IfQueries, &ifQueries)
	}
	if got.Token == nil || *got.Token != token {
		t.Fatalf("Token = %v, want %q", got.Token, token)
	}
}

func TestInjectionSpecFromAPIPreservesHTTPMatchConditions(t *testing.T) {
	headers := map[string]string{"Authorization": "Bearer test"}
	ifHeaders := map[string]string{"X-Env": "ci"}
	ifQueries := map[string]string{"debug": "1"}

	var inj apis.Injection
	if err := inj.FromHTTPInjection(apis.HTTPInjection{
		BaseURL:   "https://example.com/v1/*",
		Headers:   &headers,
		IfHeaders: &ifHeaders,
		IfQueries: &ifQueries,
		Type:      apis.HTTP,
	}); err != nil {
		t.Fatalf("FromHTTPInjection() error = %v", err)
	}

	got, err := injectionSpecFromAPI(inj)
	if err != nil {
		t.Fatalf("injectionSpecFromAPI() error = %v", err)
	}
	if got.HTTP == nil {
		t.Fatal("HTTP injection is nil")
	}
	if got.HTTP.BaseURL != "https://example.com/v1/*" {
		t.Fatalf("BaseURL = %q", got.HTTP.BaseURL)
	}
	if !reflect.DeepEqual(got.HTTP.Headers, &headers) {
		t.Fatalf("Headers = %#v, want %#v", got.HTTP.Headers, &headers)
	}
	if !reflect.DeepEqual(got.HTTP.IfHeaders, &ifHeaders) {
		t.Fatalf("IfHeaders = %#v, want %#v", got.HTTP.IfHeaders, &ifHeaders)
	}
	if !reflect.DeepEqual(got.HTTP.IfQueries, &ifQueries) {
		t.Fatalf("IfQueries = %#v, want %#v", got.HTTP.IfQueries, &ifQueries)
	}
}
