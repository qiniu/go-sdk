package sandbox

import (
	"context"
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient(&Config{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.config.APIKey != "test-api-key" {
		t.Errorf("expected APIKey 'test-api-key', got %q", client.config.APIKey)
	}
}

func TestNewClientWithEndpoint(t *testing.T) {
	client, err := NewClient(&Config{
		APIKey:   "test-api-key",
		Endpoint: "https://custom.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.config.Endpoint != "https://custom.example.com" {
		t.Errorf("expected custom endpoint, got %q", client.config.Endpoint)
	}
}

func TestNewClientDefaultEndpoint(t *testing.T) {
	client, err := NewClient(&Config{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.config.Endpoint != "" {
		t.Errorf("expected empty endpoint in config (defaults applied internally)")
	}
}

func TestClientAPI(t *testing.T) {
	client, err := NewClient(&Config{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.API() == nil {
		t.Error("expected non-nil API client")
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	customHTTP := &http.Client{}
	client, err := NewClient(&Config{
		APIKey:     "test-api-key",
		HTTPClient: customHTTP,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestAPIKeyEditor(t *testing.T) {
	editor := apiKeyEditor("test-key")

	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if err := editor(context.Background(), req); err != nil {
		t.Fatalf("editor error: %v", err)
	}
	if got := req.Header.Get("X-API-Key"); got != "test-key" {
		t.Errorf("expected X-API-Key 'test-key', got %q", got)
	}
}
