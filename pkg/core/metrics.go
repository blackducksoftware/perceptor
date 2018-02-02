package core

import (
	"net/http"
	"os"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type metrics struct {
	httpHandler                  http.Handler
	handledSuccessfulHttpRequest *prometheus.HistogramVec
}

func newMetrics() *metrics {
	m := metrics{}
	m.setup()
	m.httpHandler = prometheus.Handler()
	return &m
}

// successful http requests received

func (m *metrics) addPod(pod common.Pod) {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(0)
}

func (m *metrics) updatePod(pod common.Pod) {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(1)
}

func (m *metrics) deletePod(podName string) {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(2)
}

func (m *metrics) addImage(image common.Image) {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(3)
}

func (m *metrics) allPods(pods api.AllPods) {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(4)
}

func (m *metrics) getNextImage() {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(5)
}

func (m *metrics) postFinishedScan() {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(6)
}

func (m *metrics) getScanResults() {
	m.handledSuccessfulHttpRequest.WithLabelValues("path").Observe(7)
}

// unsuccessful http requests received

func (m *metrics) httpNotFound(request *http.Request) {
	// TODO
	log.Infof("404 when handling HTTP request to %s", request.URL.Path)
}

func (m *metrics) httpError(request *http.Request, err error) {
	// TODO
	log.Infof("error handling HTTP request to %s: %s", request.URL.Path, err.Error())
}

// model

func (m *metrics) updateModel(model Model) {
	// TODO
}

// http requests issued

// TODO

// setup

func (m *metrics) setup() {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	m.handledSuccessfulHttpRequest = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "perceptor",
			Subsystem: "perceptor-core",
			Name:      "handledSuccessfulHttpRequest",
			Help:      "requests, handled by perceptor core, which were successful",
			Buckets:   prometheus.LinearBuckets(0, 1, 8),
		},
		[]string{"path"})
	prometheus.MustRegister(m.handledSuccessfulHttpRequest)
}
