//go:build unit

package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/objects"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatLifecycleFields 通过原始 HTTP 响应验证两版 SDK 的单文件、批量查询及生命周期空值。
func TestStatLifecycleFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want [6]int64
	}{
		{
			name: "configured",
			body: `{"hash":"testhash","mimeType":"text/plain","putTime":17000000001234567,"expiration":1706000000,"transitionToIA":1701000000,"transitionToIntelligentTiering":1702000000,"transitionToArchiveIR":1703000000,"transitionToARCHIVE":1704000000,"transitionToDeepArchive":1705000000}`,
			want: [6]int64{1701000000, 1702000000, 1703000000, 1704000000, 1705000000, 1706000000},
		},
		{
			name: "zero",
			body: `{"hash":"testhash","mimeType":"text/plain","putTime":17000000001234567,"expiration":0,"transitionToIA":0,"transitionToIntelligentTiering":0,"transitionToArchiveIR":0,"transitionToARCHIVE":0,"transitionToDeepArchive":0}`,
		},
		{
			name: "omitted",
			body: `{"hash":"testhash","mimeType":"text/plain","putTime":17000000001234567}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := EncodedEntry("bucket", "object")
			missingEntry := EncodedEntry("bucket", "missing")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Reqid", "fakereqid")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/stat/"+entry:
					fmt.Fprint(w, tc.body)
				case r.Method == http.MethodPost && r.URL.Path == "/batch":
					if err := r.ParseForm(); err != nil {
						t.Errorf("parse batch form: %v", err)
						http.Error(w, "invalid form", http.StatusBadRequest)
						return
					}
					var results []string
					for _, op := range r.PostForm["op"] {
						switch strings.TrimPrefix(op, "/") {
						case "stat/" + entry:
							results = append(results, `{"code":200,"data":`+tc.body+`}`)
						case "stat/" + missingEntry:
							results = append(results, `{"code":612,"data":{"error":"no such file or directory"}}`)
						default:
							t.Errorf("unexpected operation: %s", op)
						}
					}
					fmt.Fprintf(w, "[%s]", strings.Join(results, ","))
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			credentials := auth.New("test-ak", "test-sk")
			manager := NewBucketManager(credentials, &Config{Region: &Region{RsHost: server.URL}})
			t.Run("storage/stat", func(t *testing.T) {
				info, err := manager.Stat("bucket", "object")
				require.NoError(t, err)
				assert.Equal(t, "testhash", info.Hash)
				assert.Equal(t, tc.want, [6]int64{
					info.TransitionToIA, info.TransitionToIntelligentTiering, info.TransitionToArchiveIR,
					info.TransitionToArchive, info.TransitionToDeepArchive, info.Expiration,
				})
			})
			t.Run("storage/batch", func(t *testing.T) {
				results, err := manager.Batch([]string{URIStat("bucket", "object"), URIStat("bucket", "missing")})
				require.NoError(t, err)
				require.Len(t, results, 2)
				assert.Equal(t, 200, results[0].Code)
				assert.Equal(t, 612, results[1].Code)
				assert.Equal(t, "no such file or directory", results[1].Data.Error)
				data := results[0].Data
				assert.Equal(t, "testhash", data.Hash)
				assert.Equal(t, tc.want[2], data.TransitionToArchiveIR)
				for _, field := range []struct {
					name  string
					value *int64
					want  int64
				}{
					{"TransitionToIA", data.TransitionToIA, tc.want[0]},
					{"TransitionToIntelligentTiering", data.TransitionToIntelligentTiering, tc.want[1]},
					{"TransitionToArchive", data.TransitionToArchive, tc.want[3]},
					{"TransitionToDeepArchive", data.TransitionToDeepArchive, tc.want[4]},
					{"Expiration", data.Expiration, tc.want[5]},
				} {
					if field.want == 0 {
						assert.Nil(t, field.value, field.name)
					} else {
						require.NotNil(t, field.value, field.name)
						assert.Equal(t, field.want, *field.value, field.name)
					}
				}
				assert.Nil(t, data.Status)
				assert.Nil(t, data.RestoreStatus)
			})

			objectsManager := objects.NewObjectsManager(&objects.ObjectsManagerOptions{
				Options: http_client.Options{
					Credentials: credentials,
					Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
				},
				BatchOpsExecutor: objects.NewSerialBatchOpsExecutor(nil),
			})
			object := objectsManager.Bucket("bucket").Object("object")
			t.Run("storagev2/stat", func(t *testing.T) {
				details, err := object.Stat().Call(context.Background())
				require.NoError(t, err)
				assertObjectLifecycleTimes(t, details, tc.want)
			})
			t.Run("storagev2/batch", func(t *testing.T) {
				var details *objects.ObjectDetails
				err := objectsManager.Batch(context.Background(), []objects.Operation{
					object.Stat().OnResponse(func(result *objects.ObjectDetails) { details = result }),
				}, nil)
				require.NoError(t, err)
				assertObjectLifecycleTimes(t, details, tc.want)
			})
		})
	}
}

// TestFileInfoString 验证对象信息输出的字段对齐和元数据格式。
func TestFileInfoString(t *testing.T) {
	info := FileInfo{
		Hash:                           "testhash",
		TransitionToIntelligentTiering: 1702000000,
		MetaData:                       map[string]string{"x-qn-meta-a": "b"},
	}

	result := info.String()
	if strings.Contains(result, "%!s(") {
		t.Fatalf("FileInfo.String() contains invalid format output: %s", result)
	}
	if !strings.Contains(result, "TransitionToIntelligentTiering:") {
		t.Fatalf("FileInfo.String() misses intelligent tiering field: %s", result)
	}
	for _, line := range strings.Split(strings.TrimSuffix(result, "\n"), "\n") {
		if len(line) < 35 {
			t.Fatalf("FileInfo.String() field is not aligned: %q", line)
		}
	}
}

// assertObjectLifecycleTimes 核对对象详情的生命周期时间，未设置时应保持空指针。
func assertObjectLifecycleTimes(t *testing.T, details *objects.ObjectDetails, want [6]int64) {
	t.Helper()
	require.NotNil(t, details)
	assert.Equal(t, "testhash", details.ETag)
	for i, value := range []*time.Time{
		details.TransitionToIA, details.TransitionToIntelligentTiering, details.TransitionToArchiveIR,
		details.TransitionToArchive, details.TransitionToDeepArchive, details.ExpireAt,
	} {
		if want[i] == 0 {
			assert.Nil(t, value, "field index %d", i)
		} else {
			require.NotNil(t, value, "field index %d", i)
			assert.Equal(t, want[i], value.Unix(), "field index %d", i)
		}
	}
}
