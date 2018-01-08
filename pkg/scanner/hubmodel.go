package scanner

type Project struct {
	Name     string
	Source   string
	Versions []Version
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

type PolicyStatus struct {
	OverallStatus                string
	UpdatedAt                    string
	ComponentVersionStatusCounts map[string]int
}

type ScanSummary struct {
	CreatedAt string
	Status    string
	UpdatedAt string
}
