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

	started := time.Now()
	_, err := planner.Allocate(ctx, domain.Order{
		ID: "order-3", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Allocate() error = %v, want context deadline", err)
	}
	if elapsed := time.Since(started); elapsed > 40*time.Millisecond {
		t.Fatalf("Allocate() returned after %s, want prompt cancellation", elapsed)
	}
	snapshot, _ := inventory.Snapshot(context.Background())
	if snapshot["book"] != 1 {
		t.Fatalf("available book = %d, want 1 after cancellation", snapshot["book"])
	}
}

func TestAllocateLeavesNoAuditRecordOnTimeout(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 1})
	inventory.CommitDelay = 80 * time.Millisecond
	planner := service.NewPlanner(inventory, store.NewMemoryAudit())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := planner.Allocate(ctx, domain.Order{
		ID: "order-4", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Allocate() error = %v, want context deadline", err)
	}

	report, err := planner.Report(context.Background())
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if report.AllocationCount != 0 {
		t.Fatalf("allocation count = %d, want 0 after timeout", report.AllocationCount)
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

// ctx 超时短于 CommitDelay（超时短、提交延迟长的真实场景）时的端到端不变量：
// 超时后绝不留下库存扣减或审计记录。
func TestAllocateLeavesNoReservationWhenTimeoutRacesCommitDelay(t *testing.T) {
	const iterations = 200
	for i := 0; i < iterations; i++ {
		inventory := store.NewMemoryInventory(map[string]int{"book": 1})
		inventory.CommitDelay = 20 * time.Millisecond
		planner := service.NewPlanner(inventory, store.NewMemoryAudit())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		_, err := planner.Allocate(ctx, domain.Order{
			ID: "order-race", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
		})
		cancel()

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("iteration %d: Allocate() error = %v, want context.DeadlineExceeded", i, err)
		}
		snapshot, _ := inventory.Snapshot(context.Background())
		if snapshot["book"] != 1 {
			t.Fatalf("iteration %d: available book = %d, want 1 (no deduction after timeout)", i, snapshot["book"])
		}
		report, reportErr := planner.Report(context.Background())
		if reportErr != nil {
			t.Fatalf("iteration %d: Report() error = %v", i, reportErr)
		}
		if report.AllocationCount != 0 {
			t.Fatalf("iteration %d: allocation count = %d, want 0 after timeout", i, report.AllocationCount)
		}
	}
}

// 显式取消在提交完成前发生：应尽快返回 context.Canceled，不留库存扣减或审计记录。
func TestAllocateCancelsBeforeCommitCompletes(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 1})
	inventory.CommitDelay = 80 * time.Millisecond
	planner := service.NewPlanner(inventory, store.NewMemoryAudit())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	_, err := planner.Allocate(ctx, domain.Order{
		ID: "order-cancel", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Allocate() error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(started); elapsed > 60*time.Millisecond {
		t.Fatalf("Allocate() returned after %s, want prompt cancellation", elapsed)
	}
	snapshot, _ := inventory.Snapshot(context.Background())
	if snapshot["book"] != 1 {
		t.Fatalf("available book = %d, want 1 after cancellation", snapshot["book"])
	}
	report, _ := planner.Report(context.Background())
	if report.AllocationCount != 0 {
		t.Fatalf("allocation count = %d, want 0 after cancellation", report.AllocationCount)
	}
}
