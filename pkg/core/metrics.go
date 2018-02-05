package core

import (
	"net/http"
	"os"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	httpHandler        http.Handler
	handledHTTPRequest *prometheus.CounterVec
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
	// TODO
}

// http requests issued

// TODO

// setup

func (m *metrics) setup() {
	prometheus.Unregister(prometheus.NewProcessCollector(os.Getpid(), ""))
	prometheus.Unregister(prometheus.NewGoCollector())

	m.handledHTTPRequest = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "perceptor",
		Name:        "core_http_status_codes",
		Subsystem:   "core",
		Help:        "status codes for HTTP requests handled by perceptor core",
		ConstLabels: map[string]string{},
	},
		[]string{"path", "method", "code"})
	prometheus.MustRegister(m.handledHTTPRequest)
}
