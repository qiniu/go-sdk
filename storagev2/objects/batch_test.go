//go:build unit
// +build unit

package objects

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/apis"
	"github.com/qiniu/go-sdk/v7/storagev2/apis/batch_ops"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/region"
)

func TestTopoSort(t *testing.T) {
	objectsManager := NewObjectsManager(nil)
	object1 := objectsManager.Bucket("bucket1").Object("object1")
	object2 := objectsManager.Bucket("bucket2").Object("object2")
	object3 := objectsManager.Bucket("bucket3").Object("object3")
	object4 := objectsManager.Bucket("bucket4").Object("object4")
	operations := []Operation{
		object4.Stat(),
		object1.Stat(),
		object2.Stat(),
		object3.Stat(),
	}
	assertTopoSort(t, operations, [][]Operation{{operations[0]}, {operations[1]}, {operations[2]}, {operations[3]}})
	operations = []Operation{
		object4.Stat(),
		object1.Stat(),
		object2.Stat(),
		object3.Stat(),
		object1.SetLifeCycle().DeleteAfterDays(1),
		object2.SetLifeCycle().DeleteAfterDays(1),
		object1.CopyTo("bucket2", "object2"),
		object2.CopyTo("bucket3", "object3"),
	}
	assertTopoSort(t, operations, [][]Operation{{operations[0]}, operations[1:]})
	operations = []Operation{
		object1.Stat(),
		object2.Stat(),
		object3.Stat(),
		object1.SetLifeCycle().DeleteAfterDays(1),
		object2.SetLifeCycle().DeleteAfterDays(2),
		object4.CopyTo("bucket2", "object2"),
		object2.CopyTo("bucket3", "object3"),
	}
	assertTopoSort(t, operations,
		[][]Operation{
			{operations[0], operations[3]},
			{operations[1], operations[2], operations[4], operations[5], operations[6]},
		})
	operations = []Operation{
		object4.Stat(),
		object1.Stat(),
		object2.Stat(),
		object3.Stat(),
		object1.SetLifeCycle().DeleteAfterDays(1),
		object2.SetLifeCycle().DeleteAfterDays(1),
		object1.CopyTo("bucket2", "object2"),
		object2.CopyTo("bucket3", "object3"),
		object4.CopyTo("bucket3", "object3"),
	}
	assertTopoSort(t, operations, [][]Operation{operations})
}

func assertTopoSort(t *testing.T, operations []Operation, operationsGroups [][]Operation) {
	sortedGroups, err := topoSort(operations)
	if err != nil {
		t.Fatal(err)
	}
next:
	for _, actual := range filterOperations(sortedGroups) {
		for _, expected := range operationsGroups {
			if isOperationsEqual(actual, expected) {
				continue next
			}
		}
		t.Fatalf("failed to match topo sort result")
	}
}

func isOperationsEqual(operations1, operations2 []Operation) bool {
	m := make(map[string]struct{}, len(operations1))
	for _, operation := range operations1 {
		m[operation.String()] = struct{}{}
	}
	for _, operation := range operations2 {
		if _, ok := m[operation.String()]; ok {
			delete(m, operation.String())
		} else {
			return false
		}
	}
	return true
}

func TestDoOperations(t *testing.T) {
	objectNamesDeleteCounts := make(map[string]uint)
	objectNamesResponsedCounts := make(map[string]uint)
	mux := http.NewServeMux()
	mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		responses := make([]batch_ops.OperationResponse, 0, len(r.PostForm["op"]))
		for _, op := range r.PostForm["op"] {
			if !strings.HasPrefix(op, "delete/") {
				t.Fatalf("unexpected op: %s", op)
			}
			op = strings.TrimPrefix(op, "delete/")
			entryBytes, err := base64.URLEncoding.DecodeString(op)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(entryBytes), "bucket1:") {
				t.Fatalf("unexpected op entry: %s", entryBytes)
			}
			objectName := strings.TrimPrefix(string(entryBytes), "bucket1:")
			objectNamesDeleteCounts[objectName] += 1
			responses = append(responses, batch_ops.OperationResponse{Code: 200})
		}
		respBody, err := json.Marshal(&batch_ops.Response{
			OperationResponses: responses,
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(respBody)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	objectsManager := NewObjectsManager(nil)
	bucket1 := objectsManager.Bucket("bucket1")

	operations := make([]*operation, 20)
	for i := 0; i < 20; i++ {
		objectName := fmt.Sprintf("object_%02d", i)
		operations[i] = &operation{Operation: bucket1.Object(objectName).Delete().OnResponse(func() {
			objectNamesResponsedCounts[objectName] += 1
		}).OnError(func(err error) {
			t.Fatal(err)
		})}
	}

	operations, err := doOperations(context.Background(), operations, apis.NewStorage(&http_client.Options{
		Credentials: credentials.NewCredentials("testak", "testsk"),
		Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
	}), 10, 3)
	if err != nil {
		t.Fatal(err)
	} else if len(operations) > 0 {
		t.Fatalf("unexpected operations returned")
	}
	if len(objectNamesDeleteCounts) != 20 {
		t.Fatalf("unexpected object names deleted count map")
	}
	for _, count := range objectNamesDeleteCounts {
		if count != 1 {
			t.Fatalf("unexpected objects deleted count")
		}
	}
	if len(objectNamesResponsedCounts) != 20 {
		t.Fatalf("unexpected object names responsed map")
	}
	for _, count := range objectNamesResponsedCounts {
		if count != 1 {
			t.Fatalf("unexpected objects responsed count")
		}
	}
}

func TestDoOperationsRetries(t *testing.T) {
	objectNames := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		objectNames = append(objectNames, fmt.Sprintf("object%02d", i))
	}
	objectNamesDeleteCounts := make(map[string]uint)
	objectNamesResponsedCounts := make(map[string]uint)
	mux := http.NewServeMux()
	mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		responses := make([]batch_ops.OperationResponse, 0, len(r.PostForm["op"]))
		for _, op := range r.PostForm["op"] {
			if !strings.HasPrefix(op, "delete/") {
				t.Fatalf("unexpected op: %s", op)
			}
			op = strings.TrimPrefix(op, "delete/")
			entryBytes, err := base64.URLEncoding.DecodeString(op)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(entryBytes), "bucket1:") {
				t.Fatalf("unexpected op entry: %s", entryBytes)
			}
			objectName := strings.TrimPrefix(string(entryBytes), "bucket1:")
			objectNamesDeleteCounts[objectName] += 1
			responses = append(responses, batch_ops.OperationResponse{Code: 599, Data: batch_ops.OperationResponseData{Error: "test error"}})
		}
		respBody, err := json.Marshal(&batch_ops.Response{
			OperationResponses: responses,
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(respBody)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	objectsManager := NewObjectsManager(nil)
	bucket1 := objectsManager.Bucket("bucket1")

	operations := make([]*operation, len(objectNames))
	for i, objectName := range objectNames {
		thisObjectName := objectName
		operations[i] = &operation{Operation: bucket1.Object(thisObjectName).Delete().OnResponse(func() {
			t.Fatalf("unexpected response")
		}).OnError(func(err error) {
			objectNamesResponsedCounts[thisObjectName] += 1
		})}
	}

	operations, err := doOperations(context.Background(), operations, apis.NewStorage(&http_client.Options{
		Credentials: credentials.NewCredentials("testak", "testsk"),
		Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
	}), 10, 3)
	if err != nil {
		t.Fatal(err)
	} else if len(operations) > 0 {
		t.Fatalf("unexpected operations returned")
	}
	if len(objectNamesDeleteCounts) != 20 {
		t.Fatalf("unexpected object names delete count map")
	}
	for _, count := range objectNamesDeleteCounts {
		if count != 3 {
			t.Fatalf("unexpected objects delete count")
		}
	}
	if len(objectNamesResponsedCounts) != 20 {
		t.Fatalf("unexpected object names responsed count map")
	}
	for _, count := range objectNamesResponsedCounts {
		if count != 3 {
			t.Fatalf("unexpected objects responsed count")
		}
	}
}

func TestDoOperationsDontRetry(t *testing.T) {
	objectNames := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		objectNames = append(objectNames, fmt.Sprintf("object%02d", i))
	}
	objectNamesDeleteCounts := make(map[string]uint)
	objectNamesResponsedCounts := make(map[string]uint)
	mux := http.NewServeMux()
	mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		responses := make([]batch_ops.OperationResponse, 0, len(r.PostForm["op"]))
		for _, op := range r.PostForm["op"] {
			if !strings.HasPrefix(op, "delete/") {
				t.Fatalf("unexpected op: %s", op)
			}
			op = strings.TrimPrefix(op, "delete/")
			entryBytes, err := base64.URLEncoding.DecodeString(op)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(entryBytes), "bucket1:") {
				t.Fatalf("unexpected op entry: %s", entryBytes)
			}
			objectName := strings.TrimPrefix(string(entryBytes), "bucket1:")
			objectNamesDeleteCounts[objectName] += 1
			responses = append(responses, batch_ops.OperationResponse{Code: 614})
		}
		respBody, err := json.Marshal(&batch_ops.Response{
			OperationResponses: responses,
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.Write(respBody)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	objectsManager := NewObjectsManager(nil)
	bucket1 := objectsManager.Bucket("bucket1")

	operations := make([]*operation, len(objectNames))
	for i, objectName := range objectNames {
		thisObjectName := objectName
		operations[i] = &operation{Operation: bucket1.Object(thisObjectName).Delete().OnResponse(func() {
			t.Fatalf("unexpected responsed")
		}).OnError(func(err error) {
			objectNamesResponsedCounts[thisObjectName] += 1
		})}
	}

	operations, err := doOperations(context.Background(), operations, apis.NewStorage(&http_client.Options{
		Credentials: credentials.NewCredentials("testak", "testsk"),
		Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
	}), 10, 3)
	if err != nil {
		t.Fatal(err)
	} else if len(operations) > 0 {
		t.Fatalf("unexpected operations returned")
	}
	if len(objectNamesDeleteCounts) != 20 {
		t.Fatalf("unexpected object names deleted count map")
	}
	for _, count := range objectNamesDeleteCounts {
		if count != 1 {
			t.Fatalf("unexpected objects deleted count")
		}
	}
	if len(objectNamesResponsedCounts) != 20 {
		t.Fatalf("unexpected object names responsed count map")
	}
	for _, count := range objectNamesResponsedCounts {
		if count != 1 {
			t.Fatalf("unexpected objects responsed count")
		}
	}
}

func TestRequestManagerGetOperations(t *testing.T) {
	objectsManager := NewObjectsManager(nil)
	bucket1 := objectsManager.Bucket("bucket1")
	object1 := bucket1.Object("object1")
	object2 := bucket1.Object("object2")
	object3 := bucket1.Object("object3")

	requestsManager, err := newRequestsManager(apis.NewStorage(nil), 4, 4, 4, 2, 1, 1*time.Minute, []Operation{
		object1.Stat(),
		object1.Stat(),
		object1.Stat(),
		object1.Stat(),
		object1.Stat(),
		object2.Stat(),
		object2.Stat(),
		object2.Stat(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if operations := requestsManager.takeOperations(); len(operations) != 3 {
		t.Fatalf("unexpected got operations")
	}
	if operations := requestsManager.takeOperations(); len(operations) != 5 {
		t.Fatalf("unexpected got operations")
	}

	requestsManager, err = newRequestsManager(apis.NewStorage(nil), 4, 4, 4, 2, 1, 1*time.Minute, []Operation{
		object1.Stat(),
		object1.Stat(),
		object1.Stat(),
		object1.Stat(),
		object1.Stat(),
		object2.Stat(),
		object2.Stat(),
		object2.Stat(),
		object3.Stat(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if operations := requestsManager.takeOperations(); len(operations) != 4 {
		t.Fatalf("unexpected got operations")
	}
	if operations := requestsManager.takeOperations(); len(operations) != 5 {
		t.Fatalf("unexpected got operations")
	}
}

func TestRequestManagerBatchSize(t *testing.T) {
	objectsManager := NewObjectsManager(nil)
	bucket1 := objectsManager.Bucket("bucket1")
	operations := make([]Operation, 0, 10000)
	for i := 0; i < 10000; i++ {
		operations = append(operations, bucket1.Object(fmt.Sprintf("object_%04d", i)).Stat())
	}

	const interval = 1 * time.Second
	requestsManager, err := newRequestsManager(apis.NewStorage(nil), 256, 1, 1000, 2, 1, interval, operations)
	if err != nil {
		t.Fatal(err)
	}
	defer requestsManager.done()
	time.Sleep(100 * time.Millisecond)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if size := len(requestsManager.takeOperations()); size != 256 {
		t.Fatalf("unexpected operations got, actual: %d, expected: %d", size, 256)
	}
	<-ticker.C
	requestsManager.decreaseBatchSize()
	ticker = time.NewTicker(interval)
	if size := len(requestsManager.takeOperations()); size != 256 {
		t.Fatalf("unexpected operations got, actual: %d, expected: %d", size, 256)
	}
	for i := 0; i < 3; i++ {
		<-ticker.C
	}
	if size := len(requestsManager.takeOperations()); size != 1000 {
		t.Fatalf("unexpected operations got, actual: %d, expected: %d", size, 1000)
	}
	for i := 0; i < 10; i++ {
		requestsManager.decreaseBatchSize()
	}
	ticker = time.NewTicker(interval)
	if size := len(requestsManager.takeOperations()); size != 1 {
		t.Fatalf("unexpected operations got, actual: %d, expected: %d", size, 1)
	}
	for i := 0; i < 11; i++ {
		<-ticker.C
	}
	if size := len(requestsManager.takeOperations()); size != 1000 {
		t.Fatalf("unexpected operations got, actual: %d, expected: %d", size, 1000)
	}
}

func TestWorkersManagerDoOperations573(t *testing.T) {
	objectNames := make([]string, 100000)
	for i := 0; i < 100000; i++ {
		objectNames[i] = fmt.Sprintf("object%05d", i)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		w.Header().Add("X-ReqId", "fakereqid")
		w.WriteHeader(573)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	objectsManager := NewObjectsManager(&ObjectsManagerOptions{})
	bucket1 := objectsManager.Bucket("bucket1")

	operations := make([]Operation, 100000)
	for i := 0; i < 100000; i++ {
		objectName := fmt.Sprintf("object_%05d", i)
		operations[i] = bucket1.Object(objectName).Delete()
	}

	requestsManager, err := newRequestsManager(apis.NewStorage(&http_client.Options{
		Credentials: credentials.NewCredentials("ak", "sk"),
		Regions:     &region.Region{Rs: region.Endpoints{Preferred: []string{server.URL}}},
	}), 100, 100, 100, 2, 1, 1*time.Minute, operations)
	if err != nil {
		t.Fatal(err)
	}
	defer requestsManager.done()

	workersManager := newWorkersManager(context.Background(), 10, 10, 10, 1*time.Minute, requestsManager)
	workersManager.wait()
}
