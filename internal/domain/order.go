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
	normalized := append([]Line(nil), lines...)
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].SKU < normalized[j].SKU
	})
	return normalized, nil
}
