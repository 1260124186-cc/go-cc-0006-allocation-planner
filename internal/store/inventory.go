package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

var (
	ErrUnknownSKU      = errors.New("unknown SKU")
	ErrInsufficientQty = errors.New("insufficient quantity")
)

type Inventory struct {
	mu            sync.Mutex
	available     map[string]int
	reservations  map[string]domain.Allocation
	CommitDelay   time.Duration
	SnapshotDelay time.Duration
}

func NewMemoryInventory(items map[string]int) *Inventory {
	available := make(map[string]int, len(items))
	for sku, quantity := range items {
		available[sku] = quantity
	}
	return &Inventory{
		available:    available,
		reservations: make(map[string]domain.Allocation),
	}
}

func (i *Inventory) Reserve(ctx context.Context, allocation domain.Allocation) (*domain.Allocation, error) {
	if err := waitForCommit(ctx, i.CommitDelay); err != nil {
		return nil, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, line := range allocation.Lines {
		available, ok := i.available[line.SKU]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownSKU, line.SKU)
		}
		if available < line.Quantity {
			return nil, fmt.Errorf("%w: %s", ErrInsufficientQty, line.SKU)
		}
	}
	for _, line := range allocation.Lines {
		i.available[line.SKU] -= line.Quantity
	}
	i.reservations[allocation.ID] = allocation
	copy := allocation
	return &copy, nil
}

func (i *Inventory) Release(_ context.Context, allocation domain.Allocation) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	reservation, ok := i.reservations[allocation.ID]
	if !ok {
		return fmt.Errorf("allocation %s is not reserved", allocation.ID)
	}
	for _, line := range reservation.Lines {
		i.available[line.SKU] += line.Quantity
	}
	delete(i.reservations, allocation.ID)
	return nil
}

func (i *Inventory) Snapshot(ctx context.Context) (map[string]int, error) {
	if err := waitForCommit(ctx, i.SnapshotDelay); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	i.mu.Lock()
	defer i.mu.Unlock()

	snapshot := make(map[string]int, len(i.available))
	for sku, quantity := range i.available {
		snapshot[sku] = quantity
	}
	return snapshot, nil
}

func waitForCommit(ctx context.Context, delay time.Duration) error {
	if delay == 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
