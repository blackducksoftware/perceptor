package core

import (
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

type reducer struct {
	model <-chan Model
}

// logic

func newReducer(initialModel Model,
	actions <-chan action,
	getNextImageForHubPolling <-chan func(image *common.Image),
	hubCheckResults <-chan HubImageScan,
	hubScanResults <-chan HubImageScan) *reducer {
	model := initialModel
	modelStream := make(chan Model)
	go func() {
		for {
			select {
			case nextAction := <-actions:
				model = nextAction.apply(model)
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
	return &reducer{model: modelStream}
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
