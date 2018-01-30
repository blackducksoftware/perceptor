package hub

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

func (version *Version) IsImageScanDone() bool {
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

		for _, scanSummary := range codeLocation.ScanSummaries {
			// and for each scan summary:
			switch scanSummary.Status {
			case "ERROR", "ERROR_BUILDING_BOM", "ERROR_MATCHING", "ERROR_SAVING_SCAN_DATA", "ERROR_SCANNING", "CANCELLED", "COMPLETE":
				continue
			default:
				return false
			}
		}
	}

	// log.Infof("found a project version that's done: %v", version)

	// then it's done
	return true
}
