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
