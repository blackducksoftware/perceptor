package core

import "bitbucket.org/bdsengineering/perceptor/pkg/hub"

type ImageScanResults struct {
	ScanStatus  ScanStatus
	ScanResults *hub.ImageScan
}

func NewImageScanResults() *ImageScanResults {
	return &ImageScanResults{
		ScanStatus:  ScanStatusUnknown,
		ScanResults: nil,
	}
}
