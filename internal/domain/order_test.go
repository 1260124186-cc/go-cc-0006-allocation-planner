package domain

import "testing"

func TestNormalizeLinesMergesAndSorts(t *testing.T) {
	lines, err := NormalizeLines([]Line{
		{SKU: "zinc", Quantity: 1},
		{SKU: "alpha", Quantity: 2},
		{SKU: "zinc", Quantity: 3},
	})
	if err != nil {
		t.Fatalf("NormalizeLines() error = %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	if lines[0] != (Line{SKU: "alpha", Quantity: 2}) || lines[1] != (Line{SKU: "zinc", Quantity: 4}) {
		t.Fatalf("NormalizeLines() = %#v", lines)
	}
}
