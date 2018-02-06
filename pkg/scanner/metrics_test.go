package scanner

import (
	"fmt"
	"testing"
	"time"

	"bitbucket.org/bdsengineering/perceptor/pkg/docker"
	log "github.com/sirupsen/logrus"
)

func TestMetrics(t *testing.T) {
	scanResults := make(chan ScanClientJobResults)
	httpResults := make(chan HttpResult)
	m := ScannerMetricsHandler(scanResults, httpResults)
	if m == nil {
		t.Error("expected m to be non-nil")
	}

	duration := time.Duration(4078 * time.Millisecond)
	createDuration := time.Duration(16384 * time.Millisecond)
	saveDuration := time.Duration(32768 * time.Millisecond)
	totalDuration := time.Duration(createDuration.Nanoseconds() + saveDuration.Nanoseconds())
	fileSize := 123423
	scanResults <- ScanClientJobResults{
		DockerStats: docker.ImagePullStats{
			CreateDuration: &createDuration,
			Err:            nil,
			SaveDuration:   &saveDuration,
			TotalDuration:  &totalDuration,
			TarFileSizeMBs: &fileSize,
		},
		Err:                &ScanError{Code: ErrorTypeFailedToRunJavaScanner, RootCause: fmt.Errorf("oops")},
		ScanClientDuration: &duration,
	}

	httpResults <- HttpResult{
		Path:       PathGetNextImage,
		StatusCode: 200,
	}

	message := "finished test case"
	t.Log(message)
	log.Info(message)
}
