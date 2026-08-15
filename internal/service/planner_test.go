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

type cancelAfterRecordAudit struct {
	audit  service.AuditStore
	cancel context.CancelFunc
}

func (a *cancelAfterRecordAudit) Record(ctx context.Context, allocation domain.Allocation) error {
	if err := a.audit.Record(ctx, allocation); err != nil {
		return err
	}
	a.cancel()
	return nil
}

func (a *cancelAfterRecordAudit) Remove(ctx context.Context, allocation domain.Allocation) error {
	return a.audit.Remove(ctx, allocation)
}

func (a *cancelAfterRecordAudit) OpenReport(ctx context.Context) (domain.ReportSession, error) {
	return a.audit.OpenReport(ctx)
}

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
	report, reportErr := planner.Report(context.Background())
	if reportErr != nil {
		t.Fatalf("Report() error = %v", reportErr)
	}
	if report.AllocationCount != 0 {
		t.Fatalf("audit allocation count = %d, want 0 after cancellation", report.AllocationCount)
	}
}

func TestAllocateCancelsWhileAuditReportIsOpen(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 1})
	audit := store.NewMemoryAudit()
	planner := service.NewPlanner(inventory, audit)
	session, err := audit.OpenReport(context.Background())
	if err != nil {
		t.Fatalf("OpenReport() error = %v", err)
	}
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	started := time.Now()
	go func() {
		_, err := planner.Allocate(ctx, domain.Order{
			ID: "order-audit-timeout", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
		})
		result <- err
	}()

	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Allocate() error = %v, want context deadline", err)
		}
	case <-time.After(40 * time.Millisecond):
		t.Fatal("Allocate() did not return promptly while audit report lock was held")
	}
	if elapsed := time.Since(started); elapsed > 40*time.Millisecond {
		t.Fatalf("Allocate() returned after %s, want prompt cancellation", elapsed)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("ReportSession.Close() error = %v", err)
	}

	snapshot, snapshotErr := inventory.Snapshot(context.Background())
	if snapshotErr != nil {
		t.Fatalf("Snapshot() error = %v", snapshotErr)
	}
	if snapshot["book"] != 1 {
		t.Fatalf("available book = %d, want 1 after cancellation", snapshot["book"])
	}
	report, reportErr := planner.Report(context.Background())
	if reportErr != nil {
		t.Fatalf("Report() error = %v", reportErr)
	}
	if report.AllocationCount != 0 {
		t.Fatalf("audit allocation count = %d, want 0 after cancellation", report.AllocationCount)
	}
}

func TestAllocateRollsBackWhenCanceledAfterAuditRecord(t *testing.T) {
	inventory := store.NewMemoryInventory(map[string]int{"book": 1})
	ctx, cancel := context.WithCancel(context.Background())
	audit := &cancelAfterRecordAudit{audit: store.NewMemoryAudit(), cancel: cancel}
	planner := service.NewPlanner(inventory, audit)

	_, err := planner.Allocate(ctx, domain.Order{
		ID: "order-audit-cleanup", Lines: []domain.Line{{SKU: "book", Quantity: 1}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Allocate() error = %v, want canceled context", err)
	}
	snapshot, snapshotErr := inventory.Snapshot(context.Background())
	if snapshotErr != nil {
		t.Fatalf("Snapshot() error = %v", snapshotErr)
	}
	if snapshot["book"] != 1 {
		t.Fatalf("available book = %d, want 1 after cancellation", snapshot["book"])
	}
	report, reportErr := planner.Report(context.Background())
	if reportErr != nil {
		t.Fatalf("Report() error = %v", reportErr)
	}
	if report.AllocationCount != 0 {
		t.Fatalf("audit allocation count = %d, want 0 after cancellation", report.AllocationCount)
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
