//go:build unit

package sandbox

import (
	"context"
	"net/http"
	"testing"

	"github.com/qiniu/go-sdk/v7/sandbox/internal/apis"
)

type resourceMockAPI struct {
	*mockAPI
	getResourcesFn  func(context.Context, apis.SandboxID, ...apis.RequestEditorFn) (*apis.GetSandboxResourcesResponse, error)
	patchResourceFn func(context.Context, apis.SandboxID, string, apis.PatchSandboxResourceJSONRequestBody, ...apis.RequestEditorFn) (*apis.PatchSandboxResourceResponse, error)
}

func (m *resourceMockAPI) GetSandboxResourcesWithResponse(ctx context.Context, sandboxID apis.SandboxID, editors ...apis.RequestEditorFn) (*apis.GetSandboxResourcesResponse, error) {
	return m.getResourcesFn(ctx, sandboxID, editors...)
}

func (m *resourceMockAPI) PatchSandboxResourceWithResponse(ctx context.Context, sandboxID apis.SandboxID, resourceID string, body apis.PatchSandboxResourceJSONRequestBody, editors ...apis.RequestEditorFn) (*apis.PatchSandboxResourceResponse, error) {
	return m.patchResourceFn(ctx, sandboxID, resourceID, body, editors...)
}

func TestClientGetResources(t *testing.T) {
	resourceID := "res_123"
	resource := apis.SandboxResource{}
	if err := resource.FromGitRepositoryResource(apis.GitRepositoryResource{
		MountPath:  "/workspace/repo",
		ResourceID: &resourceID,
		Type:       apis.GithubRepository,
		URL:        "https://github.com/qiniu/go-sdk.git",
	}); err != nil {
		t.Fatal(err)
	}
	mock := &resourceMockAPI{
		mockAPI: &mockAPI{},
		getResourcesFn: func(_ context.Context, sandboxID apis.SandboxID, _ ...apis.RequestEditorFn) (*apis.GetSandboxResourcesResponse, error) {
			if sandboxID != "sandbox-1" {
				t.Fatalf("unexpected sandbox ID: %s", sandboxID)
			}
			return &apis.GetSandboxResourcesResponse{
				HTTPResponse: httpResponse(http.StatusOK),
				JSON200: &struct {
					Resources []apis.SandboxResource `json:"resources"`
				}{Resources: []apis.SandboxResource{resource}},
			}, nil
		},
	}

	client := newTestClient(mock)
	resources, err := client.GetResources(context.Background(), "sandbox-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].GitRepository == nil {
		t.Fatalf("unexpected resources: %#v", resources)
	}
	if resources[0].GitRepository.ResourceID != "res_123" {
		t.Fatalf("unexpected Git resource: %#v", resources[0].GitRepository)
	}
}

func TestClientUpdateGitRepositoryResourceToken(t *testing.T) {
	mock := &resourceMockAPI{
		mockAPI: &mockAPI{},
		patchResourceFn: func(_ context.Context, sandboxID apis.SandboxID, resourceID string, body apis.PatchSandboxResourceJSONRequestBody, _ ...apis.RequestEditorFn) (*apis.PatchSandboxResourceResponse, error) {
			if sandboxID != "sandbox-1" || resourceID != "res_123" {
				t.Fatalf("unexpected resource target: %s/%s", sandboxID, resourceID)
			}
			if body.AuthorizationToken == nil || *body.AuthorizationToken != "new-token" {
				t.Fatalf("unexpected patch body: %#v", body)
			}
			return &apis.PatchSandboxResourceResponse{HTTPResponse: httpResponse(http.StatusNoContent)}, nil
		},
	}

	client := newTestClient(mock)
	if err := client.UpdateGitRepositoryResourceToken(context.Background(), "sandbox-1", "res_123", "new-token"); err != nil {
		t.Fatal(err)
	}
}

func TestClientGetResourcesAPIError(t *testing.T) {
	mock := &resourceMockAPI{
		mockAPI: &mockAPI{},
		getResourcesFn: func(context.Context, apis.SandboxID, ...apis.RequestEditorFn) (*apis.GetSandboxResourcesResponse, error) {
			return &apis.GetSandboxResourcesResponse{
				Body:         []byte(`{"message":"not found"}`),
				HTTPResponse: httpResponse(http.StatusNotFound),
				JSON404:      &apis.N404{Message: "not found"},
			}, nil
		},
	}

	client := newTestClient(mock)
	if _, err := client.GetResources(context.Background(), "sandbox-1"); err == nil {
		t.Fatal("expected API error")
	}
}
