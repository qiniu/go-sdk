//go:build unit
// +build unit

package region

import (
	"testing"
)

func TestRegion(t *testing.T) {
	region := GetRegionByID("z0", true)
	iter, err := region.EndpointsIter([]ServiceName{ServiceUp})
	if err != nil {
		t.Fatal(err)
	}
	var domain string
	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://upload.qiniup.com" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://upload-z0.qiniup.com" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://up.qiniup.com" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://up-z0.qiniup.com" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://up.qbox.me" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://up-z0.qbox.me" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); ok {
		t.Fatalf("should not get next domain")
	}

	iter, err = region.EndpointsIter([]ServiceName{ServiceUp})
	if err != nil {
		t.Fatal(err)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://upload.qiniup.com" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	iter.SwitchToAlternative()

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://up.qbox.me" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); !ok {
		t.Fatalf("should get next domain")
	} else if domain != "https://up-z0.qbox.me" {
		t.Fatalf("unexpected domain: %s", domain)
	}

	if ok := iter.Next(&domain); ok {
		t.Fatalf("should not get next domain")
	}
}
