package hub

type PolicyStatus struct {
	OverallStatus                string
	UpdatedAt                    string
	ComponentVersionStatusCounts map[string]int
}

func (ps *PolicyStatus) ViolationCount() int {
	violationCount, ok := ps.ComponentVersionStatusCounts["IN_VIOLATION"]
	if !ok {
		return 0
	}
	return violationCount
}
