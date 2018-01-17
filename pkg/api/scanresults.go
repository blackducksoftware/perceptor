package httpserver

type ScanResults struct {
	// TODO should ScannerVersion and HubServer be handled by perceiver, or supplied by perceptor?
	ScannerVersion string
	HubServer      string
	Pods           []Pod
	Images         []Image
}

func NewScanResults(scannerVersion string, hubServer string, pods []Pod, images []Image) *ScanResults {
	return &ScanResults{ScannerVersion: scannerVersion, HubServer: hubServer, Pods: pods, Images: images}
}

type Pod struct {
	Namespace        string
	Name             string
	PolicyViolations int
	Vulnerabilities  int
	OverallStatus    string
	// can't add ProjectVersionURL and ScanID because these are potentially
	// multivalued for pods
}

type Image struct {
	PolicyViolations  int
	Vulnerabilities   int
	ProjectVersionURL string
	ScanID            string
}
