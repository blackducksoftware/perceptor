package core

import (
	"sync"
	"time"

	clustermanager "bitbucket.org/bdsengineering/perceptor/pkg/clustermanager"
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	scanner "bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	log "github.com/sirupsen/logrus"
)

// Perceptor ties together a cluster manager and a hub.
// It listens to the cluster manager to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It writes vulnerabilities to pods in the cluster manager.
type Perceptor struct {
	mutex         sync.Mutex
	scannerClient scanner.ScanClientInterface
	clusterClient clustermanager.Client
	cache         VulnerabilityCache

	hubProjectName string

	// ignore the bools -- pretend like it's a set
	inProgressScanJobs map[common.Image]bool
}

// NewMockedPerceptor creates a Perceptor which uses a
// mock scanclient and mock clustermanager
func NewMockedPerceptor() (*Perceptor, error) {
	return newPerceptorHelper(scanner.NewMockHub(), clustermanager.NewMockClient()), nil
}

// NewPerceptor creates a Perceptor using the real kube client and the
// real hub client.
func NewPerceptor(clusterMasterURL string, kubeconfigPath string, username string, password string, hubHost string) (*Perceptor, error) {
	scannerClient, err := scanner.NewHubScanClient(username, password, hubHost)
	if err != nil {
		log.Errorf("unable to instantiate HubScanClient: %s", err.Error())
		return nil, err
	}
	clusterClient, err := clustermanager.NewKubeClient(clusterMasterURL, kubeconfigPath)

	if err != nil {
		log.Fatalf("unable to instantiate kubernetes client: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(scannerClient, clusterClient), nil
}

func newPerceptorHelper(scannerClient scanner.ScanClientInterface, clusterClient clustermanager.Client) *Perceptor {
	perceptor := Perceptor{
		mutex:              sync.Mutex{},
		scannerClient:      scannerClient,
		clusterClient:      clusterClient,
		cache:              *NewVulnerabilityCache(),
		hubProjectName:     "Perceptor",
		inProgressScanJobs: make(map[common.Image]bool)}

	go perceptor.startPollingClusterManagerForNewPods()
	go perceptor.startScanningImages()
	go perceptor.startPollingScanClient()
	go perceptor.startWritingPodUpdates()

	return &perceptor
}

func (perceptor *Perceptor) startPollingClusterManagerForNewPods() {
	for {
		select {
		case addPod := <-perceptor.clusterClient.PodAdd():
			perceptor.cache.AddPod(addPod.New)
			images := []string{}
			for _, cont := range addPod.New.Spec.Containers {
				images = append(images, cont.Image.Name+", "+cont.Name)
			}
			log.Infof("cluster manager event -- add pod: %v\n%v", addPod.New.Annotations, images)
		case updatePod := <-perceptor.clusterClient.PodUpdate():
			log.Infof("cluster manager event -- update pod: %v", updatePod.New.Annotations)
		case deletePod := <-perceptor.clusterClient.PodDelete():
			log.Infof("cluster manager event -- delete pod: %v", deletePod)
		}
	}
}

func (perceptor *Perceptor) startScanningImages() {
	for i := 0; ; i++ {
		select {
		case image := <-perceptor.cache.ImagesToBeScanned():
			log.Infof("should scan image %s", image)
			// TODO need to think about how to limit concurrent scans to <= 7
			// but for now, we're going to purposely block this thread so as
			// to keep the number at 1
			// TODO there seems to be a problem -- this thread gets unblocked before
			// the hub is *actually* done scanning.  So ... how do we make sure that
			// this waits until the hub is done with the previous one, before starting
			// the next one.

			perceptor.mutex.Lock()
			perceptor.inProgressScanJobs[image] = true
			perceptor.mutex.Unlock()

			err := perceptor.scannerClient.Scan(*scanner.NewScanJob(perceptor.hubProjectName, image))
			if err != nil {
				log.Errorf("error scanning image: %s", err.Error())
			}
		}
	}
}

func (perceptor *Perceptor) startPollingScanClient() {
	for {
		project, err := perceptor.scannerClient.FetchProject(perceptor.hubProjectName)

		if err != nil {
			log.Errorf("error fetching project %s: %s", perceptor.hubProjectName, err.Error())
			continue
		}

		// Check whether any jobs have been completed
		perceptor.mutex.Lock()
		images := []common.Image{}
		for image := range perceptor.inProgressScanJobs {
			images = append(images, image)
		}
		for _, image := range images {
			if project.IsImageScanDone(image) {
				delete(perceptor.inProgressScanJobs, image)
			}
		}
		perceptor.mutex.Unlock()

		// add the hub results into the cache
		perceptor.mutex.Lock()
		log.Infof("about to add scan results from project %s: %v", perceptor.hubProjectName, *project)
		err = perceptor.cache.AddScanResultsFromProject(*project)
		if err != nil {
			log.Errorf("unable to add scan result from project to cache: %s", err.Error())
		}
		perceptor.mutex.Unlock()

		// wait around for a while before checking the hub again
		time.Sleep(20 * time.Second)
		log.Info("poll for finished scans")
	}
}

func (perceptor *Perceptor) startWritingPodUpdates() {
	for {
		select {
		case update := <-perceptor.cache.ImageScanComplete():
			log.Infof("received completed image scan: %v\n", update)
			for _, pod := range update.AffectedPods {
				bdAnnotations, err := perceptor.clusterClient.GetBlackDuckPodAnnotations(pod.Namespace, pod.Name)
				if err != nil {
					log.Errorf("unable to get BlackDuckAnnotations for pod %s:%s -- %s", pod.Namespace, pod.Name, err.Error())
					continue
				}
				bdAnnotations.ImageAnnotations[update.Image] = clustermanager.ImageAnnotation{
					PolicyViolationCount: update.ScanResults.PolicyViolationCount,
					VulnerabilityCount:   update.ScanResults.VulnerabilityCount,
				}
				err = perceptor.clusterClient.SetBlackDuckPodAnnotations(pod.Namespace, pod.Name, *bdAnnotations)
				if err != nil {
					log.Errorf("unable to update BlackDuckAnnotations for pod %s:%s -- %s", pod.Namespace, pod.Name, err.Error())
				}
			}
		}
	}
}
