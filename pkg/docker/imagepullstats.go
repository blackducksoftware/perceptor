package docker

import (
	"time"
)

type ImagePullStats struct {
	CreateDuration *time.Duration
	SaveDuration   *time.Duration
	TotalDuration  *time.Duration
	TarFileSizeMBs *int
	Err            *ImagePullError
}
