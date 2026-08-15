package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

type Audit struct {
	gate      chan struct{}
	recordsMu sync.Mutex
	records   []domain.Allocation
	FailFor   map[string]error
}

func NewMemoryAudit() *Audit {
	audit := &Audit{
		gate:    make(chan struct{}, 1),
		FailFor: make(map[string]error),
	}
	audit.gate <- struct{}{}
	return audit
}

func (a *Audit) Record(ctx context.Context, allocation domain.Allocation) error {
	if err := a.acquire(ctx); err != nil {
		return err
	}
	defer a.release()

	if err := ctx.Err(); err != nil {
		return err
	}

	a.recordsMu.Lock()
	defer a.recordsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err, shouldFail := a.FailFor[allocation.OrderID]; shouldFail {
		return fmt.Errorf("record allocation: %w", err)
	}
	a.records = append(a.records, allocation)
	if err := ctx.Err(); err != nil {
		a.records = a.records[:len(a.records)-1]
		return err
	}
	return nil
}

func (a *Audit) Remove(ctx context.Context, allocation domain.Allocation) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	a.recordsMu.Lock()
	defer a.recordsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for index := len(a.records) - 1; index >= 0; index-- {
		if a.records[index].ID != allocation.ID {
			continue
		}
		a.records = append(a.records[:index], a.records[index+1:]...)
		return nil
	}
	return fmt.Errorf("allocation %s is not recorded", allocation.ID)
}

func (a *Audit) OpenReport(ctx context.Context) (domain.ReportSession, error) {
	if err := a.acquire(ctx); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		a.release()
		return nil, err
	}
	a.recordsMu.Lock()
	records := append([]domain.Allocation(nil), a.records...)
	a.recordsMu.Unlock()
	if err := ctx.Err(); err != nil {
		a.release()
		return nil, err
	}
	return &ReportSession{audit: a, records: records}, nil
}

func (a *Audit) acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-a.gate:
		return nil
	}
}

func (a *Audit) release() {
	a.gate <- struct{}{}
}

type ReportSession struct {
	audit   *Audit
	records []domain.Allocation
	closed  bool
}

func (s *ReportSession) Build() domain.Report {
	return domain.NewReport(s.records)
}

func (s *ReportSession) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	s.audit.release()
	return nil
}
