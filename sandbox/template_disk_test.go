//go:build unit

package sandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTemplateDiskParameters(t *testing.T) {
	zero, total, free := int32(0), int32(10240), int32(5120)
	for _, tt := range []struct {
		name   string
		params CreateTemplateParams
		want   map[string]any
	}{
		{"omitted", CreateTemplateParams{}, map[string]any{}},
		{"zero", CreateTemplateParams{MinDiskMB: &zero}, map[string]any{"minDiskMb": float64(0)}},
		{"positive", CreateTemplateParams{MinDiskMB: &total}, map[string]any{"minDiskMb": float64(10240)}},
		{"both", CreateTemplateParams{MinDiskMB: &total, MinFreeDiskMB: &free}, map[string]any{"minDiskMb": float64(10240), "minFreeDiskMb": float64(5120)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v3/templates", r.URL.Path)
				var body map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, tt.want, body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"templateID":"tmpl","buildID":"build"}`))
			}))
			defer server.Close()
			c, err := NewClient(&Config{Endpoint: server.URL})
			require.NoError(t, err)
			resp, err := c.CreateTemplate(context.Background(), tt.params)
			require.NoError(t, err)
			assert.Equal(t, "tmpl", resp.TemplateID)
		})
	}
}
