package core

import (
	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
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
	getNextImageForHubPolling <-chan func(image *common.Image),
	hubCheckResults <-chan HubImageScan,
	hubScanResults <-chan HubImageScan) *reducer {
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
				model = updateModelAddPod(pod, model)
				go func() {
					modelStream <- model
				}()
			case pod := <-updatePod:
				model = updateModelUpdatePod(pod, model)
				go func() {
					modelStream <- model
				}()
			case podName := <-deletePod:
				model = updateModelDeletePod(podName, model)
				go func() {
					modelStream <- model
				}()
			case image := <-addImage:
				model = updateModelAddImage(image, model)
				go func() {
					modelStream <- model
				}()
			case pods := <-allPods:
				model = updateModelUpdateAllPods(pods, model)
				go func() {
					modelStream <- model
				}()
			case continuation := <-postNextImage:
				model = updateModelNextImage(continuation, model)
				go func() {
					modelStream <- model
				}()
			case jobResults := <-finishScanClientJob:
				model = updateModelFinishedScanClientJob(jobResults, model)
				go func() {
					modelStream <- model
				}()
			case continuation := <-getNextImageForHubPolling:
				model = updateModelGetNextImageForHubPolling(continuation, model)
				go func() {
					modelStream <- model
				}()
			case imageScan := <-hubCheckResults:
				model = updateModelAddHubCheckResults(imageScan, model)
				go func() {
					modelStream <- model
				}()
			case imageScan := <-hubScanResults:
				model = updateModelAddHubScanResults(imageScan, model)
				go func() {
					modelStream <- model
				}()
			}
		}
	}()
	return &reducer{model: modelStream, imageScanStats: imageScanStats}
}

// perceivers

func updateModelAddPod(pod common.Pod, model Model) Model {
	model.AddPod(pod)
	return model
}

func updateModelUpdatePod(pod common.Pod, model Model) Model {
	model.AddPod(pod)
	return model
}

func updateModelDeletePod(podName string, model Model) Model {
	_, ok := model.Pods[podName]
	if !ok {
		log.Warnf("unable to delete pod %s, pod not found", podName)
		return model
	}
	delete(model.Pods, podName)
	return model
}

func updateModelAddImage(image common.Image, model Model) Model {
	model.AddImage(image)
	return model
}

func updateModelUpdateAllPods(pods []common.Pod, model Model) Model {
	model.Pods = map[string]common.Pod{}
	for _, pod := range pods {
		model.AddPod(pod)
	}
	return model
}

func updateModelNextImage(continuation func(image *common.Image), model Model) Model {
	log.Infof("looking for next image to scan with concurrency limit of %d, and %d currently in progress", model.ConcurrentScanLimit, model.inProgressScanCount())
	image := model.getNextImageFromScanQueue()
	continuation(image)
	return model
}

func updateModelFinishedScanClientJob(results api.FinishedScanClientJob, model Model) Model {
	newModel := model
	log.Infof("finished scan client job action: error was empty? %t, %v", results.Err == "", results.Image)
	if results.Err == "" {
		newModel.finishRunningScanClient(results.Image)
	} else {
		newModel.errorRunningScanClient(results.Image)
	}
	return newModel
}

func updateModelGetNextImageForHubPolling(continuation func(image *common.Image), model Model) Model {
	log.Infof("looking for next image to search for in hub")
	image := model.getNextImageFromHubCheckQueue()
	continuation(image)
	return model
}

func updateModelAddHubCheckResults(scan HubImageScan, model Model) Model {
	image := scan.Image

	scanResults, ok := model.Images[image]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", image.HumanReadableName())
		return model
	}

	scanResults.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan == nil {
		model.addImageToScanQueue(image)
	} else if scan.Scan.IsDone() {
		scanResults.ScanStatus = ScanStatusComplete
	} else {
		// TODO
		// it could be in the scan client stage, in the hub stage ... maybe perceptor crashed and just came back up
	}

	return model
}

func updateModelAddHubScanResults(scan HubImageScan, model Model) Model {
	image := scan.Image

	scanResults, ok := model.Images[image]
	if !ok {
		log.Warnf("expected to already have image %s, but did not", image.HumanReadableName())
		return model
	}

	scanResults.ScanResults = scan.Scan

	//	log.Infof("completing image scan of image %s ? %t", image.ShaName(), scan.Scan.IsDone())
	if scan.Scan != nil && scan.Scan.IsDone() {
		scanResults.ScanStatus = ScanStatusComplete
	}

	return model
}
