package hubapi

type CodeLocationList struct {
	TotalCount uint32         `json:"totalCount"`
	Items      []CodeLocation `json:"items"`
	Meta       Meta           `json:"_meta"`
}

type CodeLocation struct {
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	MappedProjectVersion string `json:"mappedProjectVersion"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
	Meta                 Meta   `json:"_meta"`
}

type ScanSummaryList struct {
	TotalCount uint32        `json:"totalCount"`
	Items      []ScanSummary `json:"items"`
	Meta       Meta          `json:"_meta"`
}

type ScanSummary struct {
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Meta      Meta   `json:"_meta"`
}

// I wonder if these can exist to make request as well...
// Or maybe add something to the link itself to make the request?

func (c *CodeLocation) GetScanSummariesLink() (*ResourceLink, error) {
	return c.Meta.FindLinkByRel("scans")
}

func (s *ScanSummary) GetCodeLocationLink() (*ResourceLink, error) {
	return s.Meta.FindLinkByRel("codelocation")
}
