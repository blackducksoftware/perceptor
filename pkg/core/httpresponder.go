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
	model             Model
	metricsHandler    http.Handler
	addPod            chan common.Pod
	updatePod         chan common.Pod
	deletePod         chan string
	addImage          chan common.Image
	allPods           chan []common.Pod
	postNextImage     chan func(*common.Image)
	postFinishScanJob chan api.FinishedScanClientJob
}

func NewHTTPResponder(model <-chan Model, metricsHandler http.Handler) *HTTPResponder {
	hr := HTTPResponder{
		metricsHandler:    metricsHandler,
		addPod:            make(chan common.Pod),
		updatePod:         make(chan common.Pod),
		deletePod:         make(chan string),
		addImage:          make(chan common.Image),
		allPods:           make(chan []common.Pod),
		postNextImage:     make(chan func(*common.Image)),
		postFinishScanJob: make(chan api.FinishedScanClientJob)}
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

func (hr *HTTPResponder) GetModel() string {
	jsonBytes, err := json.Marshal(hr.model)
	if err != nil {
		return fmt.Sprintf("unable to serialize model: %s", err.Error())
	}
	return string(jsonBytes)
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

func (hr *HTTPResponder) AddImage(image common.Image) {
	hr.addImage <- image
	log.Infof("handled add image %s", image.Name())
}

func (hr *HTTPResponder) UpdateAllPods(allPods api.AllPods) {
	hr.allPods <- allPods.Pods
	log.Infof("handled update all pods -- %d pods", len(allPods.Pods))
}

func (hr *HTTPResponder) GetScanResults() api.ScanResults {
	scannerVersion := "" // TODO
	hubServer := ""      // TODO
	pods := []api.Pod{}
	images := []api.Image{}
	for podName, pod := range hr.model.Pods {
		scanResults, err := hr.model.scanResults(podName)
		if err != nil {
			log.Errorf("unable to retrieve scan results for Pod %s: %s", podName, err.Error())
			continue
		}
		pods = append(pods, api.Pod{Namespace: pod.Namespace, Name: pod.Name, PolicyViolations: scanResults.PolicyViolationCount, Vulnerabilities: scanResults.VulnerabilityCount, OverallStatus: scanResults.OverallStatus})
	}
	for image, imageResults := range hr.model.Images {
		scanID := "TODO"
		projectVersionURL := "TODO"
		policyViolations := 0
		vulnerabilities := 0
		if imageResults.ScanResults != nil {
			policyViolations = imageResults.ScanResults.PolicyViolationCount
			vulnerabilities = imageResults.ScanResults.VulnerabilityCount
		}
		apiImage := api.Image{
			Name:              image.Name(),
			ScanID:            scanID,
			PolicyViolations:  policyViolations,
			Vulnerabilities:   vulnerabilities,
			ProjectVersionURL: projectVersionURL}
		images = append(images, apiImage)
	}
	return *api.NewScanResults(scannerVersion, hubServer, pods, images)
}

func (hr *HTTPResponder) GetNextImage(continuation func(nextImage api.NextImage)) {
	hr.postNextImage <- func(image *common.Image) {
		continuation(*api.NewNextImage(image))
	}
}

func (hr *HTTPResponder) PostFinishScan(job api.FinishedScanClientJob) {
	hr.postFinishScanJob <- job
	log.Infof("handled finished scan job -- %v", job)
}
