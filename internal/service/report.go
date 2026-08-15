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
	// OpenReport 持有审计锁，锁的释放委托给 Close。
	// 用 defer 保证任何返回路径（含 Snapshot 失败）都释放锁，
	// 避免一次失败永久阻塞后续报告请求。
	defer session.Close()

	if _, err := p.inventory.Snapshot(ctx); err != nil {
		return domain.Report{}, err
	}
	report := session.Build()
	return report, nil
}
