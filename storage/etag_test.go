package storage

import (
	"bytes"
	"strings"
	"testing"
)

func TestEtagV1(t *testing.T) {
	etag, err := EtagV1(bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "Fto5o-5ea0sNMlW_75VgGJCv2AcJ" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	etag, err = EtagV1(strings.NewReader("etag"))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "FpLiADEaVoALPkdb8tJEJyRTXoe_" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	etag, err = EtagV1(bytes.NewReader(generateDataOfSize(1 << 20)))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "Foyl8onxBLWeRLL5oItRJphv6i4b" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	etag, err = EtagV1(bytes.NewReader(generateDataOfSize(4 * (1 << 20))))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "FicHOveBNs5Kn9d74M3b9tI4D-8r" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	etag, err = EtagV1(bytes.NewReader(generateDataOfSize(5 * (1 << 20))))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "lg-Eb5KFCuZn-cUfj_oS2PPOU9xy" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	etag, err = EtagV1(bytes.NewReader(generateDataOfSize(8 * (1 << 20))))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "lkSKZOMToDp-EqLDVuT1pyjQssl-" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	etag, err = EtagV1(bytes.NewReader(generateDataOfSize(9 * (1 << 20))))
	if err != nil {
		t.Fatal(err)
	}
	if etag != "ljgVjMtyMsOgIySv79U8Qz4TrUO4" {
		t.Fatalf("Unexpected etag: %s", etag)
	}
}

func TestIsSignByEtagV2(t *testing.T) {

	etag := ""
	if IsSignByEtagV2(etag) {
		t.Fatalf("is Sign etag error: expect not sign by etag v2")
	}

	etag = "ns56DcSIfBFUENXjdhsJTIvl3Rcu"
	if !IsSignByEtagV2(etag) {
		t.Fatalf("is Sign etag error: expect sign by etag v2 %s", etag)
	}
}

func TestEtagV2(t *testing.T) {
	etag, err := EtagV2(bytes.NewReader([]byte("helloworld")), []int64{5, 5})
	if err != nil {
		t.Fatal(err)
	}
	if etag != "ns56DcSIfBFUENXjdhsJTIvl3Rcu" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	buf := new(bytes.Buffer)
	buf.Write(generateDataOfSize(1 << 19))
	buf.Write(generateDataOfSize(1 << 19))
	etag, err = EtagV2(buf, []int64{1 << 19, 1 << 19})
	if err != nil {
		t.Fatal(err)
	}
	if etag != "nlF4JinKEDBChmFGYbEIsZt6Gxnw" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	buf.Reset()
	buf.Write(generateDataOfSize(1 << 19))
	buf.Write(generateDataOfSize(1 << 23))
	etag, err = EtagV2(buf, []int64{1 << 19, 1 << 23})
	if err != nil {
		t.Fatal(err)
	}
	if etag != "nt82yvMNHlNgZ4H8_A_4de84mr2f" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	buf.Reset()
	buf.Write(generateDataOfSize(1 << 20))
	etag, err = EtagV2(buf, []int64{1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if etag != "Foyl8onxBLWeRLL5oItRJphv6i4b" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	buf.Reset()
	buf.Write(generateDataOfSize(9 << 20))
	etag, err = EtagV2(buf, []int64{1 << 22, 1 << 22, 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if etag != "ljgVjMtyMsOgIySv79U8Qz4TrUO4" {
		t.Fatalf("Unexpected etag: %s", etag)
	}

	buf.Reset()
	buf.Write(generateDataOfSize(1 << 20))
	etag, err = EtagV2(buf, []int64{1 << 19, 1 << 19})
	if err != nil {
		t.Fatal(err)
	}
	if etag != "nlF4JinKEDBChmFGYbEIsZt6Gxnw" {
		t.Fatalf("Unexpected etag: %s", etag)
	}
}

func generateDataOfSize(size int) []byte {
	const BLOCK_SIZE = 4096
	blockData := make([]byte, BLOCK_SIZE)

	for i := 0; i < BLOCK_SIZE; i++ {
		blockData[i] = 'b'
	}
	blockData[0] = 'A'
	blockData[BLOCK_SIZE-2] = '\r'
	blockData[BLOCK_SIZE-1] = '\n'

	buf := make([]byte, 0, size)
	rest := size
	for rest > 0 {
		addSize := rest
		if addSize > BLOCK_SIZE {
			addSize = BLOCK_SIZE
		}
		buf = append(buf, blockData[:addSize]...)
		rest -= addSize
	}
	return buf
}
