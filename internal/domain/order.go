package domain

import (
	"fmt"
	"sort"
)

// Order is a request to reserve stock for a customer-facing order.
type Order struct {
	ID    string `json:"id"`
	Lines []Line `json:"lines"`
}

type Line struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

func NormalizeLines(lines []Line) ([]Line, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("order must contain at least one line")
	}

	totalBySKU := make(map[string]int, len(lines))
	for _, line := range lines {
		if line.SKU == "" {
			return nil, fmt.Errorf("line SKU is required")
		}
		if line.Quantity <= 0 {
			return nil, fmt.Errorf("line quantity for %s must be positive", line.SKU)
		}
		totalBySKU[line.SKU] += line.Quantity
	}

	normalized := make([]Line, 0, len(totalBySKU))
	for sku, quantity := range totalBySKU {
		normalized = append(normalized, Line{SKU: sku, Quantity: quantity})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].SKU < normalized[j].SKU
	})
	return normalized, nil
}
