package core

type ImageScanResults struct {
	ScanStatus  ScanStatus
	ScanResults *ScanResults
	// Name string
}

func NewImageScanResults() *ImageScanResults {
	return &ImageScanResults{
		ScanStatus:  ScanStatusNotScanned,
		ScanResults: nil,
	}
}

type ScanResults struct {
	OverallStatus        string
	VulnerabilityCount   int
	PolicyViolationCount int
	// TODO also add:
	// scanner version
	// hub version
	// project URL
}

type ScanStatus int

const (
	ScanStatusNotScanned     ScanStatus = iota
	ScanStatusInProgress     ScanStatus = iota
	ScanStatusAnnotatingPods ScanStatus = iota // or should it be AnnotatingContainers? TODO
	ScanStatusComplete       ScanStatus = iota // TODO may need to rethink this, since we can
	// always pull from kube to check the annotations, and see if it's missing anything
	// that we have TODO
)
