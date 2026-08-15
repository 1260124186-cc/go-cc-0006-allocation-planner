package service

import (
	"context"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/domain"
)

func (p *Planner) Report(ctx context.Context) (report domain.Report, err error) {
	session, err := p.audit.OpenReport(ctx)
	if err != nil {
		return domain.Report{}, err
	}
	defer func() {
		if closeErr := session.Close(); err == nil && closeErr != nil {
			report = domain.Report{}
			err = closeErr
		}
	}()

	if _, err := p.inventory.Snapshot(ctx); err != nil {
		return domain.Report{}, err
	}
	report = session.Build()
	return report, nil
}
