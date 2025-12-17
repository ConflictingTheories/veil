package fs

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func TestPutGetObjectStream(t *testing.T) {
	tmp, err := ioutil.TempDir("", "fs-stream-test-")
	if err != nil {
		t.Fatal(err)
	}

	s := New(tmp)

	payload := "hello, veil streaming"
	r := strings.NewReader(payload)
	hash, err := s.PutObjectStream(r, "text/plain")
	if err != nil {
		t.Fatalf("PutObjectStream error: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty hash")
	}

	rc, ct, err := s.GetObjectStream(hash)
	if err != nil {
		t.Fatalf("GetObjectStream error: %v", err)
	}
	defer rc.Close()
	if ct != "text/plain" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(b) != payload {
		t.Fatalf("payload mismatch: %s", string(b))
	}
}
