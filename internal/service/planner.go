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
	lines := append([]domain.Line(nil), order.Lines...)
	if len(lines) == 0 {
		return domain.Allocation{}, fmt.Errorf("order must contain at least one line")
	}
	normalized, err := domain.NormalizeLines(lines)
	if err != nil {
		return domain.Allocation{}, err
	}

	allocation := domain.NewAllocation(order.ID, normalized, p.now())
	reservation, err := p.inventory.Reserve(ctx, allocation)
	if err != nil {
		return domain.Allocation{}, err
	}
	if reservation == nil {
		return domain.Allocation{}, ErrMissingReservation
	}
	if err := p.audit.Record(ctx, *reservation); err != nil {
		if releaseErr := p.inventory.Release(context.Background(), *reservation); releaseErr != nil {
			return domain.Allocation{}, errors.Join(err, releaseErr)
		}
		return domain.Allocation{}, err
	}
	return *reservation, nil
}
