package core

type ScanResults struct {
	OverallStatus        string
	VulnerabilityCount   int
	PolicyViolationCount int
	// TODO also add:
	// scanner version
	// hub version
	// project URL
}
