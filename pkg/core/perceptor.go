package core

import (
	"time"

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	pmetrics "bitbucket.org/bdsengineering/perceptor/pkg/metrics"
	scanner "bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	log "github.com/sirupsen/logrus"
)

type FinishedScanClientJob struct {
	image   common.Image
	results *scanner.ScanClientJobResults
	err     error
}

// Perceptor ties together: a cluster, scan clients, and a hub.
// It listens to the cluster to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It publishes vulnerabilities that the cluster can find out about.
type Perceptor struct {
	scannerClient  scanner.ScanClientInterface
	httpResponder  *HTTPResponder
	HubProjectName string
	// reducer
	reducer *reducer
	// channels
	hubScanResults      chan scanner.Project
	postNextImage       chan func(image *common.Image)
	finishScanClientJob chan FinishedScanClientJob
}

// NewMockedPerceptor creates a Perceptor which uses a mock scanclient
func NewMockedPerceptor() (*Perceptor, error) {
	return newPerceptorHelper(scanner.NewMockHub()), nil
}

// NewPerceptorFromCluster creates a Perceptor using configuration pulled from
// the cluster on which it's running.
func NewPerceptorFromCluster(username string, password string, hubHost string) (*Perceptor, error) {
	scannerClient, err := scanner.NewHubScanClient(username, password, hubHost)
	if err != nil {
		log.Errorf("unable to instantiate HubScanClient: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(scannerClient), nil
}

// NewPerceptor creates a Perceptor using a real hub client.
func NewPerceptor(username string, password string, hubHost string) (*Perceptor, error) {
	scannerClient, err := scanner.NewHubScanClient(username, password, hubHost)
	if err != nil {
		log.Errorf("unable to instantiate HubScanClient: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(scannerClient), nil
}

func newPerceptorHelper(scannerClient scanner.ScanClientInterface) *Perceptor {
	// 0. this will help with circular communication
	model := make(chan Model)
	imageScanStats := make(chan pmetrics.ImageScanStats)
	hubScanResults := make(chan scanner.Project)

	// 1. here's the responder
	httpResponder := NewHTTPResponder(model, pmetrics.MetricsHandler(imageScanStats))
	api.SetupHTTPServer(httpResponder)

	// 2. eventually, these two events will be coming in over the REST API
	postNextImage := make(chan func(image *common.Image))
	finishScanClientJob := make(chan FinishedScanClientJob)

	// 3. now for the reducer
	reducer := newReducer(*NewModel(),
		httpResponder.addPod,
		httpResponder.updatePod,
		httpResponder.deletePod,
		postNextImage,
		finishScanClientJob,
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
		scannerClient:       scannerClient,
		httpResponder:       httpResponder,
		HubProjectName:      "Perceptor",
		reducer:             reducer,
		hubScanResults:      hubScanResults,
		postNextImage:       postNextImage,
		finishScanClientJob: finishScanClientJob,
	}

	// 6. eventually, this should be in a separate container
	go perceptor.startScanningImages()

	// 7. hit the hub for results
	go perceptor.startPollingHub()

	// 8. done
	return &perceptor
}

func (perceptor *Perceptor) startScanningImages() {
	for i := 0; ; i++ {
		time.Sleep(20 * time.Second)
		log.Info("about to check for images that need to be scanned")
		go func() {
			perceptor.postNextImage <- func(image *common.Image) {
				if image == nil {
					log.Info("no images to be scanned")
					return
				}
				perceptor.scanNextImage(*image)
			}
		}()
	}
}

func (perceptor *Perceptor) scanNextImage(image common.Image) {
	log.Infof("about to start running scan client for image %s", image.Name())

	// can choose which scanner to use.
	results, err := perceptor.scannerClient.Scan(*scanner.NewScanJob(perceptor.HubProjectName, image))
	// err := perceptor.scannerClient.ScanCliSh(*scanner.NewScanJob(perceptor.HubProjectName, image))
	// err := perceptor.scannerClient.ScanDockerSh(*scanner.NewScanJob(perceptor.HubProjectName, image))

	go func() {
		perceptor.finishScanClientJob <- FinishedScanClientJob{image: image, results: results, err: err}
	}()

	if err != nil {
		log.Errorf("error scanning image %s: %s", image.Name(), err.Error())
	} else {
		log.Infof("successfully scanned image %s", image.Name())
	}
}

func (perceptor *Perceptor) startPollingHub() {
	for {
		// wait around for a while before checking the hub again
		time.Sleep(20 * time.Second)
		log.Info("poll for finished scans")

		project, err := perceptor.scannerClient.FetchProject(perceptor.HubProjectName)

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
