package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

type Audit struct {
	mu      sync.Mutex
	records []domain.Allocation
	FailFor map[string]error
}

func NewMemoryAudit() *Audit {
	return &Audit{FailFor: make(map[string]error)}
}

func (a *Audit) Record(ctx context.Context, allocation domain.Allocation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if err, shouldFail := a.FailFor[allocation.OrderID]; shouldFail {
		return fmt.Errorf("record allocation: %w", err)
	}
	a.records = append(a.records, allocation)
	return nil
}

func (a *Audit) OpenReport(ctx context.Context) (domain.ReportSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a.mu.Lock()
	return &ReportSession{audit: a, records: append([]domain.Allocation(nil), a.records...)}, nil
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
	s.audit.mu.Unlock()
	return nil
}
