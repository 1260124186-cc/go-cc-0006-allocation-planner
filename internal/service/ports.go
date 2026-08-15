package service

import (
	"context"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

type InventoryStore interface {
	Reserve(context.Context, domain.Allocation) (*domain.Allocation, error)
	Release(context.Context, domain.Allocation) error
	Snapshot(context.Context) (map[string]int, error)
}

type AuditStore interface {
	Record(context.Context, domain.Allocation) error
	OpenReport(context.Context) (domain.ReportSession, error)
}
