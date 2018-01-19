package core

type ScanResults struct {
	OverallStatus        string
	VulnerabilityCount   int
	PolicyViolationCount int
	// TODO also add:
	// scanner version
	// hub version
	// project URL
	// scan id
}

func NewScanResults() *ScanResults {
	return &ScanResults{OverallStatus: "", VulnerabilityCount: 0, PolicyViolationCount: 0}
}
