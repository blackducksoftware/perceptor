package scanner

import (
	"fmt"
	"net/http"
	"os"

	"bitbucket.org/bdsengineering/perceptor/pkg/docker"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// ScannerMetricsHandler handles http requests to get prometheus metrics
// for image scanning
func ScannerMetricsHandler(imageScanStats <-chan ScanClientJobResults, httpStats <-chan HttpResult) http.Handler {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	tarballSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "tarballsize",
			Help:      "tarball file size in MBs",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
		},
		[]string{"tarballSize"})

	durations := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "timings",
			Help:      "time durations of scanner operations",
			Buckets:   prometheus.ExponentialBuckets(0.25, 2, 20),
		},
		[]string{"stage"})

	errors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "perceptor",
		Subsystem: "scanner",
		Name:      "scannerErrors",
		Help:      "error codes from image pulling and scanning",
	}, []string{"stage", "errorName"})

	httpResults := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "scanner",
		Name:        "http_response_status_codes",
		Help:        "status codes for responses from HTTP requests issued by scanner",
		ConstLabels: map[string]string{},
	},
		[]string{"request", "code"})

	go func() {
		for {
			select {
			case stats := <-imageScanStats:
				log.Infof("got new image scan stats: %v", stats)
				if stats.ScanClientDuration != nil {
					durations.With(prometheus.Labels{"stage": "scan client"}).Observe(stats.ScanClientDuration.Seconds())
				}
				if stats.DockerStats.CreateDuration != nil {
					durations.With(prometheus.Labels{"stage": "docker create"}).Observe(stats.DockerStats.CreateDuration.Seconds())
				}
				if stats.DockerStats.SaveDuration != nil {
					durations.With(prometheus.Labels{"stage": "docker save"}).Observe(stats.DockerStats.SaveDuration.Seconds())
				}
				if stats.DockerStats.TotalDuration != nil {
					durations.With(prometheus.Labels{"stage": "docker get image total"}).Observe(stats.DockerStats.TotalDuration.Seconds())
				}
				if stats.DockerStats.TarFileSizeMBs != nil {
					tarballSize.WithLabelValues("tarballSize").Observe(float64(*stats.DockerStats.TarFileSizeMBs))
				}
				err := stats.Err
				if err != nil {
					var stage string
					var errorName string
					switch e := err.RootCause.(type) {
					case docker.ImagePullError:
						stage = "docker pull"
						errorName = e.Code.String()
					default:
						stage = "running scan client"
						errorName = err.Code.String()
					}
					errors.With(prometheus.Labels{"stage": stage, "errorName": errorName})
				}
			case httpStats := <-httpStats:
				var request string
				switch httpStats.Path {
				case PathGetNextImage:
					request = "getNextImage"
				case PathPostScanResults:
					request = "finishScan"
				}
				httpResults.With(prometheus.Labels{"request": request, "code": fmt.Sprintf("%d", httpStats.StatusCode)})
			}
		}
	}()
	prometheus.MustRegister(tarballSize)
	prometheus.MustRegister(durations)
	prometheus.MustRegister(errors)
	prometheus.MustRegister(httpResults)

	return prometheus.Handler()
}
