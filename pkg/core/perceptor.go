package core

import (
	"time"

	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	scanner "bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	log "github.com/sirupsen/logrus"
)

// Perceptor ties together a cluster and a hub.
// It listens to the cluster to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It publishes vulnerabilities that the cluster can find out about.
type Perceptor struct {
	scannerClient  scanner.ScanClientInterface
	Cache          VulnerabilityCache
	HubProjectName string
	imageScanStats chan scanner.ImageScanStats
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
	perceptor := Perceptor{
		scannerClient:  scannerClient,
		Cache:          *NewVulnerabilityCache(),
		HubProjectName: "Perceptor",
		imageScanStats: make(chan scanner.ImageScanStats)}

	go perceptor.startScanningImages()
	go perceptor.startPollingScanClient()

	return &perceptor
}

func (perceptor *Perceptor) ImageScanStats() <-chan scanner.ImageScanStats {
	return perceptor.imageScanStats
}

func (perceptor *Perceptor) addPod(pod common.Pod) bool {
	return perceptor.Cache.AddPod(pod)
}

func (perceptor *Perceptor) updatePod(pod common.Pod) {
	// return perceptor.Cache.UpdatePod(pod)
}

func (perceptor *Perceptor) deletePod(qualifiedName string) {
	perceptor.Cache.DeletePod(qualifiedName)
}

func (perceptor *Perceptor) scanNextImage() {
	concurrentScanLimit := 1
	if perceptor.Cache.inProgressScanCount() >= concurrentScanLimit {
		log.Info("max concurrent scan count reached, not starting a new scan yet")
		return
	}

	image := perceptor.Cache.getNextImageFromQueue()
	if image == nil {
		log.Info("no images in scan queue")
		return
	}

	log.Infof("about to start running scan client for image %s", image.Name())

	// can choose which scanner to use.
	stats, err := perceptor.scannerClient.Scan(*scanner.NewScanJob(perceptor.HubProjectName, *image))
	// err := perceptor.scannerClient.ScanCliSh(*scanner.NewScanJob(perceptor.HubProjectName, image))
	// err := perceptor.scannerClient.ScanDockerSh(*scanner.NewScanJob(perceptor.HubProjectName, image))
	if err != nil {
		log.Errorf("error scanning image %s: %s", image.Name(), err.Error())
		perceptor.Cache.errorRunningScanClient(*image)
	} else {
		log.Infof("successfully scanned image %s", image.Name())
		perceptor.imageScanStats <- *stats
		perceptor.Cache.finishRunningScanClient(*image)
	}
}

func (perceptor *Perceptor) startScanningImages() {
	for i := 0; ; i++ {
		time.Sleep(20 * time.Second)
		go perceptor.scanNextImage()
	}
}

func (perceptor *Perceptor) startPollingScanClient() {
	for {
		// wait around for a while before checking the hub again
		time.Sleep(5 * time.Second)
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

		// add the hub results into the cache
		log.Infof("about to add scan results from project %s: found %d versions", perceptor.HubProjectName, len(project.Versions))
		err = perceptor.Cache.AddScanResultsFromProject(*project)
		if err != nil {
			log.Errorf("unable to add scan result from project to cache: %s", err.Error())
		}
	}
}
