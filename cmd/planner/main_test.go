package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunWritesAllocation(t *testing.T) {
	var output bytes.Buffer
	input := strings.NewReader(`{"inventory":{"book":3},"order":{"id":"order-1","lines":[{"sku":"book","quantity":2}]}}`)
	if err := run([]string{"-action", "allocate"}, input, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(output.String(), `"order_id":"order-1"`) {
		t.Fatalf("run() output = %s", output.String())
	}
}
