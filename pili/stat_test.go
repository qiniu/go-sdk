//go:build integration
// +build integration

package pili

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetStatUpflow(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	// test 未设置select，是否会默认
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStatUpflow(context.Background(), GetStatUpflowRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "hour",
		},
		Where: map[string][]string{"hub": {TestHub}, "area": {"!cn", "!hk"}},
	})
	ast.Nil(err)
}

func TestManager_GroupStatUpflow(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GroupStatUpflow(context.Background(), GroupStatUpflowRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "hour",
		},
		Select: []string{"flow"},
		Where:  map[string][]string{"hub": {TestHub}},
		Group:  "hub",
	})
	ast.Nil(err)

	// test group设置
	_, err = manager.GroupStatUpflow(context.Background(), GroupStatUpflowRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "hour",
		},
		Where: map[string][]string{"hub": {TestHub}},
		Group: "test",
	})
	ast.NotNil(err)
}

func TestManager_GetStatDownflow(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStatDownflow(context.Background(), GetStatDownflowRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"flow"},
		Where:  map[string][]string{"hub": {TestHub}},
	})
	ast.Nil(err)
}

func TestManager_GroupStatDownflow(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GroupStatDownflow(context.Background(), GroupStatDownflowRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"flow"},
		Where:  map[string][]string{"hub": {TestHub}},
		Group:  "area",
	})
	ast.Nil(err)
}

func TestManager_GetStatCodec(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStatCodec(context.Background(), GetStatCodecRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"duration"},
		Where:  map[string][]string{"hub": {TestHub}, "profile": {"480p", "720p"}},
	})
	ast.Nil(err)

	// test where Validate
	_, err = manager.GetStatCodec(context.Background(), GetStatCodecRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"duration"},
		Where:  map[string][]string{"testWhere": {TestHub}},
	})
	ast.NotNil(err)

	// test select Validate
	_, err = manager.GetStatCodec(context.Background(), GetStatCodecRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"count"},
		Where:  map[string][]string{},
	})
	ast.NotNil(err)
}

func TestManager_GroupStatCodec(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GroupStatCodec(context.Background(), GroupStatCodecRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"duration"},
		Where:  map[string][]string{"hub": {"!" + TestHub}, "profile": {"480p", "720p"}},
		Group:  "profile",
	})
	ast.Nil(err)
}

func TestManager_GetStatNrop(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStatNrop(context.Background(), GetStatNropRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "5min",
		},
		Select: []string{"count"},
		Where:  map[string][]string{"hub": {"!" + TestHub}, "assured": {"false"}},
	})
	ast.Nil(err)
}

func TestManager_GroupStatNrop(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GroupStatNrop(context.Background(), GroupStatNropRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "5min",
		},
		Select: []string{"count"},
		Where:  map[string][]string{"hub": {"!" + TestHub}, "assured": {"false", "true"}},
		Group:  "assured",
	})
	ast.Nil(err)
}

func TestManager_GetStatCaster(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStatCaster(context.Background(), GetStatCasterRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"upflow", "downflow", "duration"},
		Where:  map[string][]string{"resolution": {"!480p"}},
	})
	ast.Nil(err)
}

func TestManager_GroupStatCaster(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GroupStatCaster(context.Background(), GroupStatCasterRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "day",
		},
		Select: []string{"upflow", "downflow", "duration"},
		Where:  map[string][]string{"resolution": {"!480p"}},
		Group:  "resolution",
	})
	ast.Nil(err)
}

func TestManager_GetStatPub(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetStatPub(context.Background(), GetStatPubRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "hour",
		},
		Select: []string{"duration"},
		Where:  map[string][]string{},
	})
	ast.Nil(err)
}

func TestManager_GroupStatPub(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GroupStatPub(context.Background(), GroupStatPubRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "hour",
		},
		Select: []string{"duration"},
		Where:  map[string][]string{},
		Group:  "tp",
	})
	ast.Nil(err)
}

func TestForm(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	req := GroupStatUpflowRequest{
		GetStatCommonRequest: GetStatCommonRequest{
			Begin: "20210928",
			End:   "20210930",
			G:     "hour",
		},
		Where:  map[string][]string{"hub": {TestHub, "test1", "test2"}, "area": {"cn", "hk"}},
		Group:  "group1",
		Select: []string{"flow", "flow1"},
	}
	res := Form(req)
	ast.Equal([]string{"20210928"}, res["begin"])
	ast.Equal([]string{"20210930"}, res["end"])
	ast.Equal([]string{"group1"}, res["group"])
	ast.Equal([]string{TestHub, "test1", "test2"}, res["$hub"])
	ast.Equal([]string{"cn", "hk"}, res["$area"])
	ast.Equal([]string{"flow", "flow1"}, res["select"])
}
