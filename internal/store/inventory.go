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
	now           func() time.Time
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
		now:          time.Now,
	}
}

func (i *Inventory) Reserve(ctx context.Context, allocation domain.Allocation) (*domain.Allocation, error) {
	if err := waitForCommit(ctx, i.CommitDelay); err != nil {
		return nil, err
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// context.WithTimeout 的取消由后台 goroutine 异步触发，ctx.Err() 在 deadline
	// 到期瞬间可能仍为 nil；用 deadline 时间戳做确定性比较，已超时则绝不扣减。
	if err := contextExpired(ctx, i.now); err != nil {
		return nil, err
	}

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
	// 扣减后再次确认：若 ctx 在扣减期间被取消，则立即回滚，不留库存扣减或预留
	if err := contextExpired(ctx, i.now); err != nil {
		for _, line := range allocation.Lines {
			i.available[line.SKU] += line.Quantity
		}
		return nil, err
	}
	i.reservations[allocation.ID] = allocation
	copy := allocation
	return &copy, nil
}

func (i *Inventory) Release(_ context.Context, allocation domain.Allocation) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, ok := i.reservations[allocation.ID]; !ok {
		return fmt.Errorf("allocation %s is not reserved", allocation.ID)
	}
	for _, line := range allocation.Lines {
		i.available[line.SKU] += line.Quantity
	}
	delete(i.reservations, allocation.ID)
	return nil
}

func (i *Inventory) Snapshot(ctx context.Context) (map[string]int, error) {
	if err := waitForCommit(ctx, i.SnapshotDelay); err != nil {
		return nil, err
	}
	i.mu.Lock()
	defer i.mu.Unlock()

	if err := contextExpired(ctx, i.now); err != nil {
		return nil, err
	}

	snapshot := make(map[string]int, len(i.available))
	for sku, quantity := range i.available {
		snapshot[sku] = quantity
	}
	return snapshot, nil
}

func waitForCommit(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		// timer 与 ctx.Done 同时就绪时 select 可能选中本分支，ctx 此刻是否已取消
		// 由调用方在提交前用 contextExpired 做确定性判定，这里只需返回 nil
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// contextExpired 判定 ctx 是否已取消/超时。
// context.WithTimeout 的取消由后台 goroutine 异步触发，ctx.Err() 在 deadline
// 到期瞬间可能仍为 nil，导致竞态下误提交。这里优先用 deadline 时间戳做确定性比较：
// 已过 deadline 即视为超时，不依赖异步的 ctx.Err()。手动 WithCancel 无 deadline，
// 其 cancel 是同步的，ctx.Err() 对它可靠，故回退到 ctx.Err()。
func contextExpired(ctx context.Context, now func() time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && now().After(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}
