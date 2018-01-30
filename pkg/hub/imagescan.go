package hub

// ImageScan models the results that we expect to get from the hub after
// scanning a docker image.
type ImageScan struct {
	RiskProfile                      RiskProfile
	PolicyStatus                     PolicyStatus
	ScanSummary                      ScanSummary
	CodeLocationCreatedAt            string
	CodeLocationMappedProjectVersion string
	CodeLocationName                 string
	CodeLocationType                 string
	CodeLocationURL                  string
	CodeLocationUpdatedAt            string
}

func (scan *ImageScan) IsDone() bool {
	switch scan.ScanSummary.Status {
	case "ERROR", "ERROR_BUILDING_BOM", "ERROR_MATCHING", "ERROR_SAVING_SCAN_DATA", "ERROR_SCANNING", "CANCELLED", "COMPLETE":
		return true
	default:
		return false
	}
}

func (scan *ImageScan) VulnerabilityCount() int {
	return scan.RiskProfile.HighRiskVulnerabilityCount()
}

func (scan *ImageScan) PolicyViolationCount() int {
	return scan.PolicyStatus.ViolationCount()
}

func (scan *ImageScan) OverallStatus() string {
	return scan.PolicyStatus.OverallStatus
}
