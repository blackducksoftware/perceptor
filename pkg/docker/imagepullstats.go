package docker

import (
	"time"
)

type ImagePullStats struct {
	Duration       time.Duration
	TarFileSizeMBs int
}
