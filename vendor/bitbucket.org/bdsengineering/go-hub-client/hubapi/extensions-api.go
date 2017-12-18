package hubapi

const (
	ContentTypeExtensionJSON = "application/vnd.blackducksoftware.externalextension-1+json"
)

type ExternalExtension struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	InfoURL       string `json:"infoUrl"`
	Authenticated bool   `json:"authenticated"`
	Meta          Meta   `json:"_meta"`
}
