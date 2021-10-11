// +build unit

package storage

import (
	"encoding/json"
	"fmt"
	"testing"
)

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
