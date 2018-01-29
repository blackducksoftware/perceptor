package hub

type RiskProfile struct {
	Categories       map[string]map[string]int
	BomLastUpdatedAt string
}

func (rp *RiskProfile) HighRiskVulnerabilityCount() int {
	vulnerabilities, ok := rp.Categories["VULNERABILITY"]
	if !ok {
		return 0
	}
	highCount, ok := vulnerabilities["HIGH"]
	if !ok {
		return 0
	}
	return highCount
}
