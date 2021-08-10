// +build unit

package client

import (
	"testing"
)

func TestJsonDecode(t *testing.T) {
	var ret struct{}
	body := `<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchKey</Code>
  <Message>The resource you requested does not exist</Message>
  <Resource>/mybucket/myfoto.jpg</Resource>
  <RequestId>4442587FB7D0A2F9</RequestId>
</Error>`
	err := decodeJsonFromData([]byte(body), &ret)
	if err.Error() != "invalid character '<' looking for beginning of value: "+body {
		t.Fatal("unexpected error message", err.Error())
	}
}
