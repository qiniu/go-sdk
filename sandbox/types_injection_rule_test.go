//go:build unit

package sandbox

import (
	"reflect"
	"testing"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

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
