package core

import (
	"fmt"
	"net/http"
	"os"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	httpHandler        http.Handler
	handledHTTPRequest *prometheus.CounterVec
	statusGauge        *prometheus.GaugeVec
}

func newMetrics() *metrics {
	m := metrics{}
	m.setup()
	m.httpHandler = prometheus.Handler()
	return &m
}

// successful http requests received

func (m *metrics) addPod(pod common.Pod) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "POST", "code": "200"}).Add(1)
}

func (m *metrics) updatePod(pod common.Pod) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "PUT", "code": "200"}).Add(1)
}

func (m *metrics) deletePod(podName string) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "pod", "method": "DELETE", "code": "200"}).Add(1)
}

func (m *metrics) addImage(image common.Image) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "image", "method": "POST", "code": "200"}).Add(1)
}

func (m *metrics) allPods(pods api.AllPods) {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "allpods", "method": "POST", "code": "200"}).Add(1)
}

func (m *metrics) getNextImage() {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "nextimage", "method": "POST", "code": "200"}).Add(1)
}

func (m *metrics) postFinishedScan() {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "finishedscan", "method": "POST", "code": "200"}).Add(1)
}

func (m *metrics) getScanResults() {
	m.handledHTTPRequest.With(prometheus.Labels{"path": "scanresults", "method": "GET", "code": "200"}).Add(1)
}

// unsuccessful http requests received

func (m *metrics) httpNotFound(request *http.Request) {
	path := request.URL.Path
	method := request.Method
	m.handledHTTPRequest.With(prometheus.Labels{"path": path, "method": method, "code": "404"}).Add(1)
}

func (m *metrics) httpError(request *http.Request, err error) {
	path := request.URL.Path
	method := request.Method
	m.handledHTTPRequest.With(prometheus.Labels{"path": path, "method": method, "code": "500"}).Add(1)
}

// model

func (m *metrics) updateModel(model Model) {
	// TODO may actually need to watch out for performance, and not let this get called
	// every single time the model gets updated

	// number of images in each status
	statusCounts := make(map[string]int)
	for _, imageResults := range model.Images {
		statusCounts[imageResults.ScanStatus.String()]++
	}
	for key, val := range statusCounts {
		status := fmt.Sprintf("image_status_%s", key)
		m.statusGauge.With(prometheus.Labels{"name": status}).Set(float64(val))
	}

	m.statusGauge.With(prometheus.Labels{"name": "number_of_pods"}).Set(float64(len(model.Pods)))
	m.statusGauge.With(prometheus.Labels{"name": "number_of_images"}).Set(float64(len(model.Images)))

	// TODO -- these may take more computational work
	// number of images per pod
	// number of times each image seen
	// number of images without a pod pointing to them
}

// http requests issued

// results from checking hub for completed projects (errors, unexpected things, etc.)

// TODO

// setup

func (m *metrics) setup() {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	m.handledHTTPRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Subsystem:   "core",
		Name:        "http_handled_status_codes",
		Help:        "status codes for HTTP requests handled by perceptor core",
		ConstLabels: map[string]string{},
	}, []string{"path", "method", "code"})

	m.statusGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "perceptor",
		Subsystem: "core",
		Name:      "status_gauge",
		Help:      "a gauge of statuses for perceptor core's current state",
	}, []string{"name"})

	prometheus.MustRegister(m.handledHTTPRequest)
	prometheus.MustRegister(m.statusGauge)
}
