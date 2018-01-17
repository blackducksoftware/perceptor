package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	pmetrics "bitbucket.org/bdsengineering/perceptor/pkg/metrics"
	"github.com/prometheus/common/log"
)

type HttpResponder struct {
	perceptor      *Perceptor
	metricsHandler http.Handler
}

func NewHttpResponder(perceptor *Perceptor) *HttpResponder {
	return &HttpResponder{perceptor: perceptor, metricsHandler: pmetrics.MetricsHandler(perceptor.ImageScanStats())}
}

func (hr *HttpResponder) GetMetrics(w http.ResponseWriter, r *http.Request) {
	hr.metricsHandler.ServeHTTP(w, r)
}

func (hr *HttpResponder) GetModel(w http.ResponseWriter, r *http.Request) {
	jsonBytes, err := json.Marshal(hr.perceptor)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to serialize model: %s", err.Error()), 500)
		return
	}
	jsonString := string(jsonBytes)
	fmt.Fprint(w, jsonString)
}

func (hr *HttpResponder) AddPod(pod common.Pod) {
	alreadySeenPod := !hr.perceptor.addPod(pod)
	var str string
	if alreadySeenPod {
		str = "true"
	} else {
		str = "false"
	}
	log.Infof("added pod %s -- %s, already seen = %s", pod.UID, pod.QualifiedName(), str)
}

func (hr *HttpResponder) DeletePod(qualifiedName string) {
	hr.perceptor.deletePod(qualifiedName)
	log.Infof("deleted pod %s", qualifiedName)
}

func (hr *HttpResponder) UpdatePod(pod common.Pod) {
	// TODO
	log.Errorf("action for update pod %s not yet implemented", pod.QualifiedName())
}

func (hr *HttpResponder) GetScanResults() api.ScanResults {
	scannerVersion := "" // TODO
	hubServer := ""      // TODO
	pods := []api.Pod{}
	images := []api.Image{} // TODO
	for podName, pod := range hr.perceptor.Cache.Pods {
		scanResults, err := hr.perceptor.Cache.scanResults(podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		pods = append(pods, api.Pod{Namespace: pod.Namespace, Name: pod.Name, PolicyViolations: scanResults.PolicyViolationCount, Vulnerabilities: scanResults.VulnerabilityCount, OverallStatus: scanResults.OverallStatus})
	}
	return *api.NewScanResults(scannerVersion, hubServer, pods, images)
}
