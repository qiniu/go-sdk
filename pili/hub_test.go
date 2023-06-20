//go:build integration
// +build integration

package pili

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_GetHubsList(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetHubList(context.Background())
	ast.Nil(err)
}

func TestManager_GetHubInfo(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	_, err := manager.GetHubInfo(context.Background(), GetHubInfoRequest{Hub: TestHub})
	ast.Nil(err)
}

func TestManager_HubSecurity(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.HubSecurity(context.Background(), HubSecurityRequest{
		Hub:             TestHub,
		PublishSecurity: "static",
		PublishKey:      "qiniu_static_publish_key",
	})
	ast.Nil(err)
}

func TestManager_HubHlsplus(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.HubHlsplus(context.Background(), HubHlsplusRequest{Hub: TestHub, HlsPlus: false})
	ast.Nil(err)
}

func TestManager_HubPersistence(t *testing.T) {
	if SkipTest() || TestBucket == "" {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.HubPersistence(context.Background(), HubPersistenceRequest{
		Hub:                TestHub,
		StorageBucket:      TestBucket,
		LiveDataExpireDays: 30,
	})
	ast.Nil(err)
}

func TestManager_HubSnapshot(t *testing.T) {
	if SkipTest() {
		t.SkipNow()
	}
	ast := assert.New(t)
	manager := NewManager(ManagerConfig{AccessKey: AccessKey, SecretKey: SecretKey})
	err := manager.HubSnapshot(context.Background(), HubSnapshotRequest{Hub: TestHub, SnapshotInterval: 30})
	ast.Nil(err)
}
