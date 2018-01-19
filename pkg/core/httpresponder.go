package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/common/log"
)

// HTTPResponder ...
type HTTPResponder struct {
	model          Model
	metricsHandler http.Handler
	addPod         chan common.Pod
	updatePod      chan common.Pod
	deletePod      chan string
}

func NewHTTPResponder(model <-chan Model, metricsHandler http.Handler) *HTTPResponder {
	hr := HTTPResponder{
		metricsHandler: metricsHandler,
		addPod:         make(chan common.Pod),
		updatePod:      make(chan common.Pod),
		deletePod:      make(chan string)}
	go func() {
		for {
			select {
			case m := <-model:
				hr.model = m
			}
		}
	}()
	return &hr
}

func (hr *HTTPResponder) GetMetrics(w http.ResponseWriter, r *http.Request) {
	hr.metricsHandler.ServeHTTP(w, r)
}

func (hr *HTTPResponder) GetModel(w http.ResponseWriter, r *http.Request) {
	jsonBytes, err := json.Marshal(hr.model)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to serialize model: %s", err.Error()), 500)
		return
	}
	jsonString := string(jsonBytes)
	fmt.Fprint(w, jsonString)
}

func (hr *HTTPResponder) AddPod(pod common.Pod) {
	hr.addPod <- pod
	log.Infof("handled add pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) DeletePod(qualifiedName string) {
	hr.deletePod <- qualifiedName
	log.Infof("handled delete pod %s", qualifiedName)
}

func (hr *HTTPResponder) UpdatePod(pod common.Pod) {
	hr.updatePod <- pod
	log.Infof("handled update pod %s -- %s", pod.UID, pod.QualifiedName())
}

func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	scannerVersion := "" // TODO
	hubServer := ""      // TODO
	pods := []api.Pod{}
	images := []api.Image{} // TODO
	for podName, pod := range hr.model.Pods {
		scanResults, err := hr.model.scanResults(podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		pods = append(pods, api.Pod{Namespace: pod.Namespace, Name: pod.Name, PolicyViolations: scanResults.PolicyViolationCount, Vulnerabilities: scanResults.VulnerabilityCount, OverallStatus: scanResults.OverallStatus})
	}
	return *api.NewScanResults(scannerVersion, hubServer, pods, images)
}
