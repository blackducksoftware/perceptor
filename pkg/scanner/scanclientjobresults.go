package scanner

import "time"

type ScanClientJobResults struct {
	PullDuration       *time.Duration
	TarFileSizeMBs     *int
	ScanClientDuration *time.Duration
	Err                *ScanError
}
