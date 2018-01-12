package scanner

import (
	"time"

	pdocker "bitbucket.org/bdsengineering/perceptor/pkg/docker"
)

type ImageScanStats struct {
	PullStats    pdocker.ImagePullStats
	ScanDuration time.Duration
}
