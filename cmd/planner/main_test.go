package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
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

// 同一订单重复填写相同 SKU 时，应按数量汇总成一行再进入分配流程。
func TestRunMergesDuplicateSKULines(t *testing.T) {
	var output bytes.Buffer
	input := strings.NewReader(`{"inventory":{"book":5},"order":{"id":"order-2","lines":[{"sku":"book","quantity":1},{"sku":"book","quantity":2}]}}`)
	if err := run([]string{"-action", "allocate"}, input, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	var allocation domain.Allocation
	if err := json.Unmarshal(output.Bytes(), &allocation); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, output.String())
	}
	if len(allocation.Lines) != 1 || allocation.Lines[0].SKU != "book" || allocation.Lines[0].Quantity != 3 {
		t.Fatalf("allocation lines = %#v, want single merged book line of 3", allocation.Lines)
	}
}

// 正常订单（无重复 SKU）的输出行数与数量不应被改动。
func TestRunPreservesDistinctSKUOrder(t *testing.T) {
	var output bytes.Buffer
	input := strings.NewReader(`{"inventory":{"book":5,"pen":3},"order":{"id":"order-3","lines":[{"sku":"book","quantity":2},{"sku":"pen","quantity":1}]}}`)
	if err := run([]string{"-action", "allocate"}, input, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	var allocation domain.Allocation
	if err := json.Unmarshal(output.Bytes(), &allocation); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, output.String())
	}
	if len(allocation.Lines) != 2 {
		t.Fatalf("allocation lines = %#v, want 2 distinct lines", allocation.Lines)
	}
	if allocation.Lines[0] != (domain.Line{SKU: "book", Quantity: 2}) {
		t.Fatalf("first line = %#v, want book/2", allocation.Lines[0])
	}
	if allocation.Lines[1] != (domain.Line{SKU: "pen", Quantity: 1}) {
		t.Fatalf("second line = %#v, want pen/1", allocation.Lines[1])
	}
}

