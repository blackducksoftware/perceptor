package api

import (
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
)

type FinishedScanClientJob struct {
	Image   common.Image
	Results *ScanClientJobResults
	Err     string
}
