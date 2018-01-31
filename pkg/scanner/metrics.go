package scanner

import (
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

	pullDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "pullduration",
			Help:      "pull duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.25, 2, 20),
		},
		[]string{"pullDurationSeconds"})

	scanDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "metrics",
			Name:      "scanduration",
			Help:      "scan duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
		},
		[]string{"scanDurationSeconds"})

	errors := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "errors",
			Help:      "error codes from image scanning",
			Buckets:   []float64{ErrorTypeFailedToRunJavaScanner, ErrorTypeUnableToPullDockerImage},
		}, []string{"errorCode"})

	dockerErrors := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "docker errors",
			Help:      "error codes from pulling images from docker",
			Buckets:   prometheus.LinearBuckets(float64(docker.ErrorTypeUnableToCreateImage), 1, int(docker.ErrorTypeUnableToGetFileStats)+1),
		}, []string{"dockerErrorCode"})

	getNextImageHTTPResults := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "get next image HTTP results",
			Help:      "HTTP status codes from asking perceptor for images",
			Buckets:   prometheus.LinearBuckets(0, 1, 1000),
		}, []string{"statusCode"})

	postFinishedScanHTTPResults := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "scanner",
			Name:      "post finished scan HTTP results",
			Help:      "HTTP status codes from posting scan results to perceptor",
			Buckets:   prometheus.LinearBuckets(0, 1, 1000),
		}, []string{"statusCode"})

	go func() {
		for {
			select {
			case stats := <-imageScanStats:
				log.Infof("got new image scan stats: %v", stats)
				if stats.ScanClientDuration != nil {
					scanDuration.WithLabelValues("scanDurationSeconds").Observe(float64(stats.ScanClientDuration.Seconds()))
				}
				if stats.PullDuration != nil {
					pullDuration.WithLabelValues("pullDurationSeconds").Observe(float64(stats.PullDuration.Seconds()))
				}
				if stats.TarFileSizeMBs != nil {
					tarballSize.WithLabelValues("tarballSize").Observe(float64(*stats.TarFileSizeMBs))
				}
				err := stats.Err
				if err != nil {
					errors.WithLabelValues("errorCode").Observe(float64(err.Code))
					imagePullError, ok := err.RootCause.(docker.ImagePullError)
					if ok {
						dockerErrors.WithLabelValues("dockerErrorCode").Observe(float64(imagePullError.Code))
					}
				}
			case httpStats := <-httpStats:
				switch httpStats.Path {
				case PathGetNextImage:
					getNextImageHTTPResults.WithLabelValues("statusCode").Observe(float64(httpStats.StatusCode))
				case PathPostScanResults:
					postFinishedScanHTTPResults.WithLabelValues("statusCode").Observe(float64(httpStats.StatusCode))
				}
			}
		}
	}()
	prometheus.MustRegister(tarballSize)
	prometheus.MustRegister(pullDuration)
	prometheus.MustRegister(scanDuration)
	prometheus.MustRegister(errors)
	prometheus.MustRegister(dockerErrors)

	return prometheus.Handler()
}
