package core

import (
	"fmt"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/hub"
	pmetrics "bitbucket.org/bdsengineering/perceptor/pkg/metrics"
	log "github.com/sirupsen/logrus"
)

type reducer struct {
	model          <-chan Model
	imageScanStats <-chan pmetrics.ImageScanStats
	// hubStats       <-chan hubScanResults
	// httpStats      <-chan httpStats
	// errorStats     <-chan errorStats
}

// logic

func newReducer(initialModel Model,
	addPod <-chan common.Pod,
	updatePod <-chan common.Pod,
	deletePod <-chan string,
	addImage <-chan common.Image,
	allPods <-chan []common.Pod,
	postNextImage <-chan func(image *common.Image),
	finishScanClientJob <-chan api.FinishedScanClientJob,
	hubScanResults <-chan hub.Project) *reducer {
	var err error
	model := initialModel
	modelStream := make(chan Model)
	imageScanStats := make(chan pmetrics.ImageScanStats)
	// hubStats := make(chan hubScanResults)
	// httpStats := make(chan httpStats)
	// errorStats := make(chan errorStats)
	go func() {
		for {
			select {
			case pod := <-addPod:
				model, err = updateModelAddPod(pod, model)
				if err != nil {
					log.Errorf("unable to add pod %s: %s", pod.QualifiedName(), err.Error())
				}
				go func() {
					modelStream <- model
				}()
			case pod := <-updatePod:
				model, err = updateModelUpdatePod(pod, model)
				if err != nil {
					log.Errorf("unable to update pod %s: %s", pod.QualifiedName(), err.Error())
				}
				go func() {
					modelStream <- model
				}()
			case podName := <-deletePod:
				model, err = updateModelDeletePod(podName, model)
				if err != nil {
					log.Errorf("unable to delete pod %s: %s", podName, err.Error())
				}
				go func() {
					modelStream <- model
				}()
			case image := <-addImage:
				model, err = updateModelAddImage(image, model)
				if err != nil {
					log.Errorf("unable to add image %s: %s", image.Name(), err.Error())
				}
				go func() {
					modelStream <- model
				}()
			case pods := <-allPods:
				model, err = updateModelUpdateAllPods(pods, model)
				if err != nil {
					log.Errorf("unable to update all pods: %s", err.Error())
				}
				go func() {
					modelStream <- model
				}()
			case continuation := <-postNextImage:
				model, err = updateModelNextImage(continuation, model)
				if err != nil {
					log.Errorf("unable to get next image for scanning: %s", err.Error())
				}
				go func() {
					modelStream <- model
				}()
			case jobResults := <-finishScanClientJob:
				model, err = updateModelFinishedScanClientJob(jobResults, model)
				if err != nil {
					log.Errorf("unable to add finished scan client job results for image %s: %s", jobResults.Image.Name(), err.Error())
				}
				go func() {
					modelStream <- model
				}()
				if jobResults.Results != nil {
					go func() {
						imageScanStats <- pmetrics.ImageScanStats{
							PullDuration:   jobResults.Results.PullDuration,
							ScanDuration:   jobResults.Results.ScanClientDuration,
							TarFileSizeMBs: jobResults.Results.TarFileSizeMBs}
					}()
				}
			case project := <-hubScanResults:
				model, err = updateModelAddHubScanResults(project, model)
				if err != nil {
					log.Errorf("unable to add hub scan results: %s", err.Error())
				}
				go func() {
					modelStream <- model
				}()
			}
		}
	}()
	return &reducer{model: modelStream, imageScanStats: imageScanStats}
}

// perceivers

func updateModelAddPod(pod common.Pod, model Model) (Model, error) {
	model.AddPod(pod)
	return model, nil
}

func updateModelUpdatePod(pod common.Pod, model Model) (Model, error) {
	model.AddPod(pod)
	return model, nil
}

func updateModelDeletePod(podName string, model Model) (Model, error) {
	_, ok := model.Pods[podName]
	if !ok {
		return model, fmt.Errorf("unable to delete pod %s, pod not found", podName)
	}
	delete(model.Pods, podName)
	return model, nil
}

func updateModelAddImage(image common.Image, model Model) (Model, error) {
	model.AddImage(image)
	return model, nil
}

func updateModelUpdateAllPods(pods []common.Pod, model Model) (Model, error) {
	model.Pods = map[string]common.Pod{}
	for _, pod := range pods {
		model.AddPod(pod)
	}
	return model, nil
}

func updateModelNextImage(continuation func(image *common.Image), model Model) (Model, error) {
	log.Infof("looking for next image to scan with concurrency limit of %d, and %d currently in progress", model.ConcurrentScanLimit, model.inProgressScanCount())
	image := model.getNextImageFromQueue()
	continuation(image)
	return model, nil
}

func updateModelFinishedScanClientJob(results api.FinishedScanClientJob, model Model) (Model, error) {
	newModel := model
	if results.Err == nil {
		newModel.finishRunningScanClient(results.Image)
	} else {
		newModel.errorRunningScanClient(results.Image)
	}
	return newModel, nil
}

func updateModelAddHubScanResults(project hub.Project, model Model) (Model, error) {
	newModel := model
	for _, version := range project.Versions {
		err := addScanResult(&newModel, version)
		if err != nil {
			// if there's an error, don't make any changes
			return model, err
		}
	}
	return newModel, nil
}

func addScanResult(model *Model, version hub.Version) error {
	image := common.Image(version.VersionName)

	// add scan results into cache
	scanResults, ok := model.Images[image]
	if !ok {
		return fmt.Errorf("expected to already have image %s, but did not", image.Name())
	}

	if scanResults.ScanResults == nil {
		scanResults.ScanResults = NewScanResults()
	}

	scanResults.ScanResults.VulnerabilityCount = version.RiskProfile.HighRiskVulnerabilityCount()
	scanResults.ScanResults.OverallStatus = version.PolicyStatus.OverallStatus
	scanResults.ScanResults.PolicyViolationCount = version.PolicyStatus.ViolationCount()

	log.Infof("completing image scan of version %s ? %t", version.VersionName, version.IsImageScanDone())
	if version.IsImageScanDone() {
		scanResults.ScanStatus = ScanStatusComplete
	}

	return nil
}
