package metrics

import (
	"time"
)

type ImageScanStats struct {
	PullDuration   time.Duration
	ScanDuration   time.Duration
	TarFileSizeMBs int
}
