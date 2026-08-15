package domain

import (
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
	quantities := make(map[string]int, len(lines))
	for _, line := range lines {
		quantities[line.SKU] += line.Quantity
	}

	normalized := make([]Line, 0, len(quantities))
	for sku, quantity := range quantities {
		normalized = append(normalized, Line{SKU: sku, Quantity: quantity})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].SKU < normalized[j].SKU
	})
	return normalized, nil
}
