package scanner

type Project struct {
	Name     string
	Source   string
	Versions []Version
}

type Version struct {
	CodeLocations   []CodeLocation
	RiskProfile     RiskProfile
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

type ScanSummary struct {
	CreatedAt string
	Status    string
	UpdatedAt string
}
