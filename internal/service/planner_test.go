package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service"
	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/store"
)

func TestAllocateMergesDuplicateLines(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 5})
	planner := service.NewPlanner(inventory, store.NewMemoryAudit())

	allocation, err := planner.Allocate(context.Background(), domain.Order{
		ID: "order-1",
		Lines: []domain.Line{
			{SKU: "book", Quantity: 1},
			{SKU: "book", Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if len(allocation.Lines) != 1 || allocation.Lines[0].Quantity != 3 {
		t.Fatalf("allocation lines = %#v, want one merged line", allocation.Lines)
	}
}

func TestAllocateReleasesStockWhenAuditFails(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 2})
	audit := store.NewMemoryAudit()
	audit.FailFor["order-2"] = errors.New("audit offline")
	planner := service.NewPlanner(inventory, audit)

	_, err := planner.Allocate(context.Background(), domain.Order{
		ID: "order-2", Lines: []domain.Line{{SKU: "book", Quantity: 2}},
	})
	if err == nil {
		t.Fatal("Allocate() error = nil, want audit error")
	}
	snapshot, snapshotErr := inventory.Snapshot(context.Background())
	if snapshotErr != nil {
		t.Fatalf("Snapshot() error = %v", snapshotErr)
	}
	if snapshot["book"] != 2 {
		t.Fatalf("available book = %d, want 2", snapshot["book"])
	}
}

func TestAllocateHonorsCanceledContext(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 1})
	inventory.CommitDelay = 80 * time.Millisecond
	planner := service.NewPlanner(inventory, store.NewMemoryAudit())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := planner.Allocate(ctx, domain.Order{
		ID: "order-3", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Allocate() error = %v, want context deadline", err)
	}
	snapshot, _ := inventory.Snapshot(context.Background())
	if snapshot["book"] != 1 {
		t.Fatalf("available book = %d, want 1 after cancellation", snapshot["book"])
	}
}

func TestReportReleasesAuditSessionOnFailedSnapshot(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 1})
	inventory.SnapshotDelay = 80 * time.Millisecond
	planner := service.NewPlanner(inventory, store.NewMemoryAudit())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, err := planner.Report(ctx); err == nil {
		t.Fatal("Report() error = nil, want canceled context")
	}
	if _, err := planner.Report(context.Background()); err != nil {
		t.Fatalf("second Report() error = %v, want unlocked audit session", err)
	}
}
