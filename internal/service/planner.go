package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

var ErrMissingReservation = errors.New("inventory did not return a reservation")

type Planner struct {
	inventory InventoryStore
	audit     AuditStore
	now       func() time.Time
}

func NewPlanner(inventory InventoryStore, audit AuditStore) *Planner {
	return &Planner{inventory: inventory, audit: audit, now: time.Now}
}

func (p *Planner) Allocate(ctx context.Context, order domain.Order) (domain.Allocation, error) {
	if order.ID == "" {
		return domain.Allocation{}, fmt.Errorf("order ID is required")
	}
	lines, err := domain.NormalizeLines(order.Lines)
	if err != nil {
		return domain.Allocation{}, err
	}

	allocation := domain.NewAllocation(order.ID, lines, p.now())
	reservation, err := p.inventory.Reserve(ctx, allocation)
	if err != nil {
		return domain.Allocation{}, err
	}
	if reservation == nil {
		return domain.Allocation{}, ErrMissingReservation
	}
	if err := p.audit.Record(ctx, *reservation); err != nil {
		return domain.Allocation{}, err
	}
	return *reservation, nil
}
