//go:build integration
// +build integration

package storage

import (
	"encoding/json"
	"fmt"
	clientV1 "github.com/qiniu/go-sdk/v7/client"
	"strings"
	"testing"
)

func TestIndexPage(t *testing.T) {
	err := bucketManager.TurnOnIndexPage("not-exist")
	errInfo := fmt.Sprintf("%v", err)
	if !strings.Contains(errInfo, "no such bucket") {
		t.Fatalf("turn on not exist bucket index page should error:%v", err)
	}

	err = bucketManager.TurnOffIndexPage("not-exist")
	errInfo = fmt.Sprintf("%v", err)
	if !strings.Contains(errInfo, "no such bucket") {
		t.Fatalf("turn off not exist bucket index page should error:%v", err)
	}

	err = bucketManager.TurnOnIndexPage(testBucket)
	if err != nil {
		t.Fatalf("turn on index page error:%v", err)
	}

	err = bucketManager.TurnOffIndexPage(testBucket)
	if err != nil {
		t.Fatalf("turn off index page error:%v", err)
	}
}

func TestUcBucketEventRule(t *testing.T) {

	rule := &BucketEventRule{
		Name:        "name_value",
		Prefix:      "prefix_value",
		Suffix:      "suffix_value",
		Event:       []string{"event_01_value", "event_02_value"},
		CallbackURL: []string{"callback_url_01_value", "callback_url_02_value"},
		AccessKey:   "access_key_value",
		Host:        "host_value",
	}

	ruleData, err := json.Marshal(rule)
	if err != nil {
		t.Fatal(err)
	}

	ruleString := string(ruleData)
	fmt.Println(ruleString)

	ruleNew := &BucketEventRule{}
	err = json.Unmarshal(ruleData, ruleNew)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBucketTag(t *testing.T) {
	clientV1.DeepDebugInfo = true

	tagKey := "test-tag"
	tagValue := "tag-can-delete"
	err := bucketManager.SetTagging(testBucket, map[string]string{
		tagKey: tagValue,
	})
	if err != nil {
		t.Fatalf("set tag error:%s", err)
	}

	tags, err := bucketManager.GetTagging(testBucket)
	if err != nil {
		t.Fatalf("get tag error:%s", err)
	}
	if tags[tagKey] != tagValue {
		t.Fatalf("get tag value error:%s", tags)
	}

	err = bucketManager.ClearTagging(testBucket)
	if err != nil {
		t.Fatalf("clear tag error:%s", err)
	}

	tags, err = bucketManager.GetTagging(testBucket)
	if err != nil {
		t.Fatalf("get tag after clean error:%s", err)
	}
	if len(tags) > 0 {
		t.Fatal("tag should b empty  after clean")
	}
}
