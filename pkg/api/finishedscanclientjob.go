package api

import (
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/scanner"
)

type FinishedScanClientJob struct {
	Image   common.Image
	Results *scanner.ScanClientJobResults
	Err     string
}
