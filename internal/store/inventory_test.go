package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

func TestReserveUnknownSKUReturnsError(t *testing.T) {
	inventory := NewMemoryInventory(map[string]int{"book": 1})
	_, err := inventory.Reserve(context.Background(), domain.NewAllocation("order-1", []domain.Line{{SKU: "pen", Quantity: 1}}, time.Now()))
	if !errors.Is(err, ErrUnknownSKU) {
		t.Fatalf("Reserve() error = %v, want unknown SKU", err)
	}
}

func TestReleaseRestoresAvailability(t *testing.T) {
	inventory := NewMemoryInventory(map[string]int{"book": 2})
	allocation := domain.NewAllocation("order-1", []domain.Line{{SKU: "book", Quantity: 2}}, time.Now())
	reservation, err := inventory.Reserve(context.Background(), allocation)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if err := inventory.Release(context.Background(), *reservation); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	snapshot, _ := inventory.Snapshot(context.Background())
	if snapshot["book"] != 2 {
		t.Fatalf("available book = %d, want 2", snapshot["book"])
	}
}

// ctx 超时短于 CommitDelay（超时短、提交延迟长的真实场景）时，ctx 取消的异步性
// 会让 select 偶尔选中 timer 分支、而此刻 ctx.Err() 尚为 nil，修复前会继续扣减库存
// 并返回成功。这里反复触发该竞态，断言绝不发生。
func TestReserveRejectsWhenContextExpiresNearCommitDelay(t *testing.T) {
	const iterations = 200
	for i := 0; i < iterations; i++ {
		inventory := NewMemoryInventory(map[string]int{"book": 1})
		inventory.CommitDelay = 20 * time.Millisecond
		allocation := domain.NewAllocation("order-race", []domain.Line{{SKU: "book", Quantity: 1}}, time.Now())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		_, err := inventory.Reserve(ctx, allocation)
		cancel()

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("iteration %d: Reserve() error = %v, want context.DeadlineExceeded", i, err)
		}
		snapshot, _ := inventory.Snapshot(context.Background())
		if snapshot["book"] != 1 {
			t.Fatalf("iteration %d: available book = %d, want 1 (no deduction after timeout)", i, snapshot["book"])
		}
	}
}
