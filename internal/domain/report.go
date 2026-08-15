package domain

type Report struct {
	AllocationCount int `json:"allocation_count"`
	ReservedUnits   int `json:"reserved_units"`
}

// ReportSession keeps an audit read lock while a report is assembled.
type ReportSession interface {
	Build() Report
	Close() error
}

func NewReport(allocations []Allocation) Report {
	report := Report{AllocationCount: len(allocations)}
	for _, allocation := range allocations {
		for _, line := range allocation.Lines {
			report.ReservedUnits += line.Quantity
		}
	}
	return report
}
