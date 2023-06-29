//go:build integration
// +build integration

package pili

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetStreamsList(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStreamsList(context.Background(), GetStreamListRequest{Hub: TestHub, LiveOnly: false, Limit: 10})
	ast.Nil(err)

	_, err = manager.GetStreamsList(context.Background(), GetStreamListRequest{Hub: TestHub, LiveOnly: false, Limit: 99999})
	ast.NotNil(err)
}

func TestManager_GetStreamBaseInfo(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStreamBaseInfo(context.Background(), GetStreamBaseInfoRequest{Hub: TestHub, Stream: TestStream})
	ast.Nil(err)
}

func TestManager_StreamDisable(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.StreamDisable(context.Background(), StreamDisabledRequest{
		Hub:                 TestHub,
		Stream:              TestStream,
		DisablePeriodSecond: 60,
	})
	ast.Nil(err)

	err = manager.StreamDisable(context.Background(), StreamDisabledRequest{
		Hub:          TestHub,
		Stream:       TestStream,
		DisabledTill: 0,
	})
	ast.Nil(err)
}

func TestManager_GetStreamLiveStatus(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStreamLiveStatus(context.Background(), GetStreamLiveStatusRequest{Hub: TestHub, Stream: TestStream})
	if err != nil {
		ast.EqualError(err, "no live")
	}
}

func TestManager_BatchGetStreamLiveStatus(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.BatchGetStreamLiveStatus(context.Background(), BatchGetStreamLiveStatusRequest{
		Hub:   TestHub,
		Items: []string{TestStream},
	})
	ast.Nil(err)
}

func TestManager_GetStreamHistory(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStreamHistory(context.Background(), GetStreamHistoryRequest{
		Hub:    TestHub,
		Stream: TestStream,
		Start:  0,
		End:    0,
	})
	ast.Nil(err)
}

func TestManager_StreamSaveas(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.StreamSaveas(context.Background(), StreamSaveasRequest{
		Hub:        TestHub,
		Stream:     TestStream,
		Start:      1632796442,
		End:        1632810842,
		Fname:      "test",
		ExpireDays: -1,
	})
	if err != nil {
		ast.EqualError(err, "no data")
	}
}

func TestManager_StreamSnapshot(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	ret, err := manager.StreamSnapshot(context.Background(), StreamSnapshotRequest{
		Hub:             TestHub,
		Stream:          TestStream,
		Time:            1632810081,
		Fname:           "test",
		DeleteAfterDays: 1,
	})
	if err == nil {
		ast.Equal("test", ret.Fname)
	}
}

func TestManager_StreamConverts(t *testing.T) {
	if SkipTest() || TestStream == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.StreamConverts(context.Background(), StreamConvertsRequest{
		Hub:      TestHub,
		Stream:   TestStream,
		Converts: []string{"480p", "720p"},
	})
	ast.Nil(err)
}

func TestEncodeStream(t *testing.T) {
	ast := assert.New(t)
	str := encodeStream("test0")
	ast.Equal(str, "dGVzdDA=")
}
