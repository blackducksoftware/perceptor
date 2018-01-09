package clustermanager

// BlackDuckAnnotations describes the data model for pod annotation.
type BlackDuckAnnotations struct {
	// TODO remove KeyVals, this is just for testing, to be able
	// to jam random stuff somewhere
	KeyVals              map[string]string
	PolicyViolationCount int
	VulnerabilityCount   int
	OverallStatus        string
}

func NewBlackDuckAnnotations(policyViolationCount int, vulnerabilityCount int, overallStatus string) *BlackDuckAnnotations {
	return &BlackDuckAnnotations{
		PolicyViolationCount: policyViolationCount,
		VulnerabilityCount:   vulnerabilityCount,
		OverallStatus:        overallStatus,
		KeyVals:              make(map[string]string),
	}
}

func (bda *BlackDuckAnnotations) hasPolicyViolations() bool {
	return bda.PolicyViolationCount > 0
}

func (bda *BlackDuckAnnotations) hasVulnerabilities() bool {
	return bda.VulnerabilityCount > 0
}
