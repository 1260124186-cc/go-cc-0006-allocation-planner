package transport

import (
	"strings"
	"testing"
)

func TestDecodeRequestRejectsUnknownFields(t *testing.T) {
	_, err := DecodeRequest(strings.NewReader(`{"inventory":{"book":1},"unexpected":true}`))
	if err == nil {
		t.Fatal("DecodeRequest() error = nil, want unknown field error")
	}
}
