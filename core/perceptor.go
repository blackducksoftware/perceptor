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
	scannerClient      scanner.ScanClientInterface
	clusterClient      clustermanager.Client
	cache              VulnerabilityCache
	inProgressScanJobs map[string]string // map of projectName to image. for now,
	// we're doing one project per image scan, but that'll most likely change
}

// NewMockedPerceptor creates a Perceptor which uses a
// mock scanclient and mock clustermanager
func NewMockedPerceptor() (*Perceptor, error) {
	return newPerceptorHelper(scanner.NewMockHub(), clustermanager.NewMockClient()), nil
}

// NewPerceptor creates a Perceptor using the real kube client and the
// real hub client.
func NewPerceptor(clusterMasterURL string, kubeconfigPath string, username string, password string, hubHost string, pathToScanner string) (*Perceptor, error) {
	scannerClient, err := scanner.NewHubScanClient(username, password, hubHost, pathToScanner)
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
		inProgressScanJobs: make(map[string]string)}

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
				images = append(images, cont.Image+", "+cont.Name)
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
			log.Infof("about to look for project %s", projectName)

			project, err := perceptor.scannerClient.FetchProject(projectName)
			if err != nil {
				log.Errorf("error fetching project %s: %s", projectName, err.Error())
				continue
			}
			image, ok := perceptor.inProgressScanJobs[projectName]
			if !ok {
				log.Errorf("expected to find key %s in inProgressScanJobs map", projectName)
				continue
			}
			// Did we find the project, and does it have a version with a code
			// location with a scan summary which is complete?
			if project == nil {
				continue
			}
			if len(project.Versions) == 0 {
				continue
			}
			version := project.Versions[0]
			if len(version.CodeLocations) == 0 {
				continue
			}

			// if there's at least 1 code location
			// and for each code location:
			//   there's at least 1 scan summary
			//   and for each scan summary:
			//     the status is complete
			// then it's done
			if len(version.CodeLocations) == 0 {
				continue
			}

			isDone := true
			for _, codeLocation := range version.CodeLocations {
				if len(codeLocation.ScanSummaries) == 0 {
					isDone = false
					break
				}
				scanSummary := codeLocation.ScanSummaries[0]
				if scanSummary.Status != "COMPLETE" {
					isDone = false
					break
				}
			}

			if !isDone {
				continue
			}

			log.Infof("about to add scan results from project %s: %v\n\n", projectName, *project)
			delete(perceptor.inProgressScanJobs, projectName)
			err = perceptor.cache.AddScanResult(image, *project)
			if err != nil {
				log.Errorf("unable to add scan result from project to cache: %s", err.Error())
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
