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
