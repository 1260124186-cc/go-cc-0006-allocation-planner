package service

import (
	"context"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

func (p *Planner) Report(ctx context.Context) (domain.Report, error) {
	session, err := p.audit.OpenReport(ctx)
	if err != nil {
		return domain.Report{}, err
	}
	defer session.Close()

	if _, err := p.inventory.Snapshot(ctx); err != nil {
		return domain.Report{}, err
	}
	return session.Build(), nil
}
