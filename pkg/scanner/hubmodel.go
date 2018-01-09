package scanner

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

type Project struct {
	Name     string
	Source   string
	Versions []Version
}

func (project *Project) IsImageScanDone(image common.Image) bool {
	for _, version := range project.Versions {
		if version.VersionName != image.Name() {
			continue
		}

		// if there's at least 1 code location
		if len(version.CodeLocations) == 0 {
			return false
		}

		// and for each code location:
		for _, codeLocation := range version.CodeLocations {
			// there's at least 1 scan summary
			if len(codeLocation.ScanSummaries) == 0 {
				return false
			}
			scanSummary := codeLocation.ScanSummaries[0]
			// and for each scan summary:
			switch scanSummary.Status {
			// the status is complete (or canceled, or some kind of error)
			case "ERROR", "ERROR_BUILDING_BOM", "ERROR_MATCHING", "ERROR_SAVING_SCAN_DATA", "ERROR_SCANNING", "CANCELLED", "COMPLETE":
				continue
			default:
				return false
			}
		}

		// then it's done
		return true
	}

	return false
}

type Version struct {
	CodeLocations   []CodeLocation
	RiskProfile     RiskProfile
	PolicyStatus    PolicyStatus
	Distribution    string
	Nickname        string
	VersionName     string
	ReleasedOn      string
	ReleaseComments string
	Phase           string
}

type CodeLocation struct {
	ScanSummaries        []ScanSummary
	CreatedAt            string
	MappedProjectVersion string
	Name                 string
	CodeLocationType     string
	Url                  string
	UpdatedAt            string
}

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

type ScanSummary struct {
	CreatedAt string
	Status    string
	UpdatedAt string
}
