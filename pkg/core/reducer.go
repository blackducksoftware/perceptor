package core

import (
	"errors"
	"fmt"

	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	pmetrics "bitbucket.org/bdsengineering/perceptor/pkg/metrics"
	"bitbucket.org/bdsengineering/perceptor/pkg/scanner"
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
	postNextImage <-chan func(image *common.Image),
	finishScanClientJob <-chan FinishedScanClientJob,
	hubScanResults <-chan scanner.Project) *reducer {
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
					log.Errorf("unable to add finished scan client job results for image %s: %s", jobResults.image.Name(), err.Error())
				}
				go func() {
					modelStream <- model
				}()
				if jobResults.results != nil {
					go func() {
						imageScanStats <- pmetrics.ImageScanStats{
							PullDuration:   jobResults.results.PullDuration,
							ScanDuration:   jobResults.results.ScanClientDuration,
							TarFileSizeMBs: jobResults.results.TarFileSizeMBs}
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
	_, ok := model.Pods[pod.QualifiedName()]
	if ok {
		return model, fmt.Errorf("attempted to add pod %s, but pod name was already present", pod.QualifiedName())
	}
	log.Infof("about to add pod: UID %s, qualified name %s", pod.UID, pod.QualifiedName())
	for _, newCont := range pod.Containers {
		_, hasImage := model.Images[newCont.Image]
		if !hasImage {
			addedImage := NewImageScanResults()
			model.Images[newCont.Image] = addedImage
			log.Infof("adding image %s to image scan queue", newCont.Image)
			model.addImageToQueue(newCont.Image)
		} else {
			log.Infof("not adding image %s to image scan queue, already have in cache", newCont.Image)
		}
	}
	log.Infof("done adding containers+images from pod %s -- %s", pod.UID, pod.QualifiedName())
	model.Pods[pod.QualifiedName()] = pod
	return model, nil
}

func updateModelUpdatePod(pod common.Pod, model Model) (Model, error) {
	return model, fmt.Errorf("update actions are not yet implemented")
}

func updateModelDeletePod(podName string, model Model) (Model, error) {
	_, ok := model.Pods[podName]
	if !ok {
		return model, fmt.Errorf("unable to delete pod %s, pod not found", podName)
	}
	delete(model.Pods, podName)
	return model, nil
}

func updateModelNextImage(continuation func(image *common.Image), model Model) (Model, error) {
	concurrentScanLimit := 1
	log.Infof("looking for next image to scan with concurrency limit of %d, and %d currently in progress", concurrentScanLimit, model.inProgressScanCount())
	if model.inProgressScanCount() >= concurrentScanLimit {
		log.Info("max concurrent scan count reached, not starting a new scan yet")
		continuation(nil)
		return model, nil
	}

	if len(model.ImageScanQueue) == 0 {
		log.Info("scan queue empty, not starting a new scan")
		continuation(nil)
		return model, nil
	}

	first := model.ImageScanQueue[0]
	results := model.safeGet(first)
	if results.ScanStatus != ScanStatusInQueue {
		continuation(nil)
		message := fmt.Sprintf("can not start scanning image %s, status is not InQueue (%d)", first.Name(), results.ScanStatus)
		log.Errorf(message)
		return model, errors.New(message)
	}

	log.Infof("about to scan image %s", first.Name())
	continuation(&first)
	results.ScanStatus = ScanStatusRunningScanClient
	model.ImageScanQueue = model.ImageScanQueue[1:]
	return model, nil
}

func updateModelFinishedScanClientJob(results FinishedScanClientJob, model Model) (Model, error) {
	newModel := model
	if results.err == nil {
		newModel.finishRunningScanClient(results.image)
	} else {
		newModel.errorRunningScanClient(results.image)
	}
	return newModel, nil
}

func updateModelAddHubScanResults(project scanner.Project, model Model) (Model, error) {
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

func addScanResult(model *Model, version scanner.Version) error {
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
