package core

import (
	"fmt"
	"sync"
	"time"

	clustermanager "bitbucket.org/bdsengineering/perceptor/clustermanager"
	scanner "bitbucket.org/bdsengineering/perceptor/scanner"
	log "github.com/sirupsen/logrus"
)

// Perceptor ties together a cluster manager and a hub.
// It listens to the cluster manager to learn about new pods.
// It keeps track of pods, containers, images, and scan results in a model.
// It has the hub scan images that have never been seen before.
// It grabs the scan results from the hub and adds them to its model.
// It writes vulnerabilities to pods in the cluster manager.
type Perceptor struct {
	mutex              sync.Mutex
	scannerClient      scanner.HubScanClient // TODO use interface type instead?
	clusterClient      clustermanager.Client // TODO use interface type instead?
	cache              VulnerabilityCache
	inProgressScanJobs map[string]string // map of projectName to image. for now,
	// we're doing one project per image scan, but that'll most likely change

}

// NewMockedPerceptor creates a Perceptor which uses a
// mock scanclient and mock clustermanager
func NewMockedPerceptor() (*Perceptor, error) {
	// TODO
	return nil, nil
}

func NewPerceptor() (*Perceptor, error) {
	scannerClient, err := scanner.NewHubScanClient("sysadmin", "blackduck", "localhost")
	if err != nil {
		log.Errorf("unable to instantiate HubScanClient: %s", err.Error())
		return nil, err
	}
	clusterClient, err := clustermanager.NewKubeClient()

	if err != nil {
		log.Fatalf("unable to instantiate kubernetes client: %s", err.Error())
		return nil, err
	}

	cache := NewVulnerabilityCache()

	perceptor := Perceptor{
		mutex:         sync.Mutex{},
		scannerClient: *scannerClient,
		clusterClient: clusterClient,
		cache:         *cache}

	go perceptor.startPollingClusterManagerForNewPods()
	go perceptor.startScanningImages()
	go perceptor.startPollingScanClient()
	go perceptor.startWritingPodUpdates()

	return &perceptor, nil
}

func (perceptor *Perceptor) startPollingClusterManagerForNewPods() {
	for {
		select {
		case addPod := <-perceptor.clusterClient.PodAdd():
			perceptor.cache.AddPod(addPod.New)
			log.Infof("cluster manager event -- add pod: %v", addPod.New.Annotations)
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
			projectName := fmt.Sprintf("my-%s-project-%d", image, i)

			perceptor.mutex.Lock()
			perceptor.inProgressScanJobs[projectName] = image
			perceptor.mutex.Unlock()

			err := perceptor.scannerClient.Scan(*scanner.NewScanJob(projectName, image))
			if err != nil {
				log.Errorf("error scanning image: %s", err.Error())
			}
		}
	}
}

func (perceptor *Perceptor) startPollingScanClient() {
	for {
		perceptor.mutex.Lock()
		keys := []string{}
		for key, _ := range perceptor.inProgressScanJobs {
			keys = append(keys, key)
		}

		for _, projectName := range keys {
			project := perceptor.scannerClient.FetchProject(projectName)
			image, ok := perceptor.inProgressScanJobs[projectName]
			if !ok {
				log.Errorf("expected to find key %s in inProgressScanJobs map", projectName)
				continue
			}
			if project != nil {
				// TODO check whether the project (no wait, I mean scan job?) is actually done
				delete(perceptor.inProgressScanJobs, projectName)
				err := perceptor.cache.AddScanResult(image, *project)
				log.Infof("found project %s", projectName)
				if err != nil {
					log.Errorf("unable to add scan result from project to cache: %s", err.Error())
				}
			} else {
				log.Infof("did not find project %s", projectName)
			}
		}
		perceptor.mutex.Unlock()

		time.Sleep(10 * time.Second)
		log.Info("poll for finished scans")
	}
}

func (perceptor *Perceptor) startWritingPodUpdates() {
	for {
		select {
		case update := <-perceptor.cache.ImageScanComplete():
			fmt.Printf("update: %v\n", update)
			// perceptor.clusterClient.GetBlackDuckPodAnnotations(pod)
			// TODO implement
		}
	}
}
