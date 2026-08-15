package domain

import "time"

type Allocation struct {
	ID        string `json:"id"`
	OrderID   string `json:"order_id"`
	Lines     []Line `json:"lines"`
	CreatedAt string `json:"created_at"`
}

func NewAllocation(orderID string, lines []Line, now time.Time) Allocation {
	return Allocation{
		ID:        "allocation-" + orderID,
		OrderID:   orderID,
		Lines:     append([]Line(nil), lines...),
		CreatedAt: now.UTC().Format(time.RFC3339),
	}
}
