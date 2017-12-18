package hubapi

type ComplexLicense struct {
	Name           string           `json:"name"`
	Ownership      string           `json:"ownership"`
	CodeSharing    string           `json:"codeSharing"`
	LicenseType    string           `json:"type"`
	LicenseDisplay string           `json:"licenseDisplay"`
	Licenses       []ComplexLicense `json:"licenses"`
	License        string           `json:"license"` // License URL
}

type License struct {
	Name        string `json:"name"`
	Ownership   string `json:"ownership"`
	CodeSharing string `json:"codeSharing"`
	Meta        Meta   `json:"_meta"`
}
