package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	pmetrics "bitbucket.org/bdsengineering/perceptor/pkg/metrics"
)

type HttpResponder struct {
	perceptor      *Perceptor
	metricsHandler http.Handler
}

func NewHttpResponder(perceptor *Perceptor) *HttpResponder {
	return &HttpResponder{perceptor: perceptor, metricsHandler: pmetrics.MetricsHandler(perceptor.ImageScanStats())}
}

// func (ht *HttpResponder) metrics(w http.ResponseWriter, r *http.Request)
func (hr *HttpResponder) Metrics(w http.ResponseWriter, r *http.Request) {
	hr.metricsHandler.ServeHTTP(w, r)
}

func (hr *HttpResponder) Model(w http.ResponseWriter, r *http.Request) {
	jsonBytes, err := json.Marshal(hr.perceptor)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to serialize model: %s", err.Error()), 500)
		return
	}
	jsonString := string(jsonBytes)
	fmt.Fprint(w, jsonString)
}
