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
	merged := make(map[string]int, len(lines))
	order := make([]string, 0, len(lines))
	for _, line := range lines {
		if line.SKU == "" {
			return nil, fmt.Errorf("line SKU is required")
		}
		if line.Quantity <= 0 {
			return nil, fmt.Errorf("line quantity for %s must be positive", line.SKU)
		}
		if _, exists := merged[line.SKU]; !exists {
			order = append(order, line.SKU)
		}
		merged[line.SKU] += line.Quantity
	}
	sort.Strings(order)
	normalized := make([]Line, 0, len(order))
	for _, sku := range order {
		normalized = append(normalized, Line{SKU: sku, Quantity: merged[sku]})
	}
	return normalized, nil
}
