package clustermanager

// BlackDuckAnnotations describes the data model for pod annotation.
type BlackDuckAnnotations struct {
	// TODO remove KeyVals, this is just for testing, to be able
	// to jam random stuff somewhere
	KeyVals          map[string]string
	ImageAnnotations map[string]ImageAnnotation
}

func NewBlackDuckAnnotations() *BlackDuckAnnotations {
	return &BlackDuckAnnotations{
		ImageAnnotations: make(map[string]ImageAnnotation),
		KeyVals:          make(map[string]string),
	}
}

type ImageAnnotation struct {
	PolicyViolationCount int
	VulnerabilityCount   int
}

func (ia *ImageAnnotation) hasPolicyViolations() bool {
	return ia.PolicyViolationCount > 0
}

func (ia *ImageAnnotation) hasVulnerabilities() bool {
	return ia.VulnerabilityCount > 0
}
