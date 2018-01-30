package core

import (
	"sync"
	"time"

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/hub"
	pmetrics "bitbucket.org/bdsengineering/perceptor/pkg/metrics"
	log "github.com/sirupsen/logrus"
)

// Perceptor ties together: a cluster, scan clients, and a hub.
// It listens to the cluster to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It publishes vulnerabilities that the cluster can find out about.
type Perceptor struct {
	hubClient     hub.FetcherInterface
	httpResponder *HTTPResponder
	// reducer
	reducer *reducer
	// channels
	checkNextImageInHub chan func(image *common.Image)
	hubCheckResults     chan HubImageScan
	hubScanResults      chan HubImageScan
	inProgressHubScans  []common.Image
}

// NewMockedPerceptor creates a Perceptor which uses a mock scanclient
func NewMockedPerceptor() (*Perceptor, error) {
	return newPerceptorHelper(hub.NewMockHub()), nil
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(username string, password string, hubHost string) (*Perceptor, error) {
	baseURL := "https://" + hubHost
	hubClient, err := hub.NewFetcher(username, password, baseURL)
	if err != nil {
		log.Errorf("unable to instantiate hub Fetcher: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(hubClient), nil
}

func newPerceptorHelper(hubClient hub.FetcherInterface) *Perceptor {
	// 0. this will help with circular communication
	model := make(chan Model)
	imageScanStats := make(chan pmetrics.ImageScanStats)
	hubScanResults := make(chan HubImageScan)
	hubCheckResults := make(chan HubImageScan)
	checkNextImageInHub := make(chan func(image *common.Image))

	// 1. here's the responder
	httpResponder := NewHTTPResponder(model, pmetrics.MetricsHandler(imageScanStats))
	api.SetupHTTPServer(httpResponder)

	concurrentScanLimit := 1

	// 3. now for the reducer
	reducer := newReducer(*NewModel(concurrentScanLimit),
		httpResponder.addPod,
		httpResponder.updatePod,
		httpResponder.deletePod,
		httpResponder.addImage,
		httpResponder.allPods,
		httpResponder.postNextImage,
		httpResponder.postFinishScanJob,
		checkNextImageInHub,
		hubCheckResults,
		hubScanResults)

	// 5. instantiate perceptor
	perceptor := Perceptor{
		hubClient:           hubClient,
		httpResponder:       httpResponder,
		reducer:             reducer,
		checkNextImageInHub: checkNextImageInHub,
		hubCheckResults:     hubCheckResults,
		hubScanResults:      hubScanResults,
		inProgressHubScans:  []common.Image{},
	}

	// 4. close the circle
	go func() {
		for {
			select {
			case nextModel := <-reducer.model:
				perceptor.inProgressHubScans = nextModel.inProgressHubScans()
				model <- nextModel
			case nextImageScanStats := <-reducer.imageScanStats:
				imageScanStats <- nextImageScanStats
			}
		}
	}()

	// 7. hit the hub for results
	go perceptor.startCheckingForImagesInHub()
	go perceptor.startPollingHubForCompletedScans()

	// 8. done
	return &perceptor
}

func (perceptor *Perceptor) startPollingHubForCompletedScans() {
	for {
		time.Sleep(20 * time.Second)

		for _, image := range perceptor.inProgressHubScans {
			scan, err := perceptor.hubClient.FetchScanFromImage(image)
			if err != nil {
				log.Errorf("unable to fetch image scan for image %s: %s", image.HubProjectName(), err.Error())
			} else {
				if scan == nil {
					log.Infof("unable to find image scan for image %s, found nil", image.HubProjectName())
				} else {
					log.Infof("found image scan for image %s: %v", image.HubProjectName(), *scan)
				}
				perceptor.hubScanResults <- HubImageScan{Image: image, Scan: scan}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (perceptor *Perceptor) startCheckingForImagesInHub() {
	for {
		var wg sync.WaitGroup
		wg.Add(1)
		var image *common.Image
		perceptor.checkNextImageInHub <- func(i *common.Image) {
			image = i
			wg.Done()
		}
		wg.Wait()

		if image != nil {
			scan, err := perceptor.hubClient.FetchScanFromImage(*image)
			if err != nil {
				log.Errorf("unable to fetch image scan for image %s: %s", image.HubProjectName(), err.Error())
			} else {
				if scan == nil {
					log.Infof("unable to find image scan for image %s, found nil", image.HubProjectName())
				} else {
					log.Infof("found image scan for image %s: %v", image.HubProjectName(), *scan)
				}
				perceptor.hubCheckResults <- HubImageScan{Image: *image, Scan: scan}
			}
			time.Sleep(1 * time.Second)
		} else {
			// slow down the chatter if we didn't find something
			time.Sleep(20 * time.Second)
		}
	}
}
