package metrics

import (
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// MetricsHandler handles http requests to get prometheus metrics
func MetricsHandler(imageScanStats <-chan ImageScanStats) http.Handler {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	tarballSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "metrics",
			Name:      "tarballsize",
			Help:      "tarball file size in MBs",
			Buckets:   prometheus.ExponentialBuckets(1, 2, 15),
		},
		[]string{"tarballSize"})

	pullDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "metrics",
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

	go func() {
		for {
			select {
			case stats := <-imageScanStats:
				log.Infof("got new image scan stats: scan duration: %d, pull duration %d, tar file size %d",
					int(stats.ScanDuration.Seconds()),
					int(stats.PullDuration.Seconds()),
					int(stats.TarFileSizeMBs))
				tarballSize.WithLabelValues("tarballSize").Observe(float64(stats.TarFileSizeMBs))
				pullDuration.WithLabelValues("pullDurationSeconds").Observe(float64(stats.PullDuration.Seconds()))
				scanDuration.WithLabelValues("scanDurationSeconds").Observe(float64(stats.ScanDuration.Seconds()))
				continue
			}
		}
	}()
	prometheus.MustRegister(tarballSize)
	prometheus.MustRegister(pullDuration)
	prometheus.MustRegister(scanDuration)

	return prometheus.Handler()
}
