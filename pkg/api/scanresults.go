package httpserver

type ScanResults struct {
	// TODO should ScannerVersion and HubServer be handled by perceiver, or supplied by perceptor?
	ScannerVersion string
	HubServer      string
	//
	Pods []struct {
		Namespace        string
		Name             string
		PolicyViolations int
		Vulnerabilities  int
		OverallStatus    string
		// can't add ProjectVersionURL and ScanID because these are potentially
		// multivalued for pods
	}
	//
	Images []struct {
		PolicyViolations  int
		Vulnerabilities   int
		ProjectVersionURL string
		ScanID            string
	}
}
