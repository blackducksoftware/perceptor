package core

import (
	"time"

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
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
	hubClient      hub.FetcherInterface
	httpResponder  *HTTPResponder
	HubProjectName string
	// reducer
	reducer *reducer
	// channels
	hubScanResults      chan hub.Project
	finishScanClientJob chan api.FinishedScanClientJob
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
	hubScanResults := make(chan hub.Project)

	// 1. here's the responder
	httpResponder := NewHTTPResponder(model, pmetrics.MetricsHandler(imageScanStats))
	api.SetupHTTPServer(httpResponder)

	concurrentScanLimit := 1

	// 2. eventually, these two events will be coming in over the REST API
	finishScanClientJob := make(chan api.FinishedScanClientJob)

	// 3. now for the reducer
	reducer := newReducer(*NewModel(concurrentScanLimit),
		httpResponder.addPod,
		httpResponder.updatePod,
		httpResponder.deletePod,
		httpResponder.postNextImage,
		httpResponder.postFinishScanJob,
		hubScanResults)

	// 4. close the circle
	go func() {
		for {
			select {
			case nextModel := <-reducer.model:
				model <- nextModel
			case nextImageScanStats := <-reducer.imageScanStats:
				imageScanStats <- nextImageScanStats
			}
		}
	}()

	// 5. instantiate perceptor
	perceptor := Perceptor{
		hubClient:           hubClient,
		httpResponder:       httpResponder,
		HubProjectName:      hub.PerceptorProjectName,
		reducer:             reducer,
		hubScanResults:      hubScanResults,
		finishScanClientJob: finishScanClientJob,
	}

	// 7. hit the hub for results
	go perceptor.startPollingHub()

	// 8. done
	return &perceptor
}

func (perceptor *Perceptor) startPollingHub() {
	for {
		// wait around for a while before checking the hub again
		time.Sleep(20 * time.Second)
		log.Info("poll for finished scans")

		project, err := perceptor.hubClient.FetchProjectByName(perceptor.HubProjectName)

		if err != nil {
			log.Errorf("error fetching project %s: %s", perceptor.HubProjectName, err.Error())
			continue
		}

		if project == nil {
			log.Errorf("cannot find project %s", perceptor.HubProjectName)
			continue
		}

		log.Infof("about to add scan results from project %s: found %d versions", perceptor.HubProjectName, len(project.Versions))
		go func() {
			perceptor.hubScanResults <- *project
		}()
	}
}
