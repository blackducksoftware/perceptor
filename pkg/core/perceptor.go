package core

import (
	"time"

	clustermanager "bitbucket.org/bdsengineering/perceptor/pkg/clustermanager"
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
	scannerClient  scanner.ScanClientInterface
	clusterClient  clustermanager.Client
	Cache          VulnerabilityCache
	HubProjectName string
	imageScanStats chan scanner.ImageScanStats
}

// NewMockedPerceptor creates a Perceptor which uses a
// mock scanclient and mock clustermanager
func NewMockedPerceptor() (*Perceptor, error) {
	return newPerceptorHelper(scanner.NewMockHub(), clustermanager.NewMockClient()), nil
}

// NewPerceptorFromCluster creates a Perceptor using configuration pulled from
// the cluster on which it's running.
func NewPerceptorFromCluster(username string, password string, hubHost string) (*Perceptor, error) {
	scannerClient, err := scanner.NewHubScanClient(username, password, hubHost)
	if err != nil {
		log.Errorf("unable to instantiate HubScanClient: %s", err.Error())
		return nil, err
	}
	clusterClient, err := clustermanager.NewKubeClientFromCluster()

	if err != nil {
		log.Errorf("unable to instantiate kubernetes client: %s", err.Error())
		return nil, err
	}

	return newPerceptorHelper(scannerClient, clusterClient), nil
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
		scannerClient:  scannerClient,
		clusterClient:  clusterClient,
		Cache:          *NewVulnerabilityCache(),
		HubProjectName: "Perceptor",
		imageScanStats: make(chan scanner.ImageScanStats)}

	go perceptor.startPollingClusterManagerForNewPods()
	go perceptor.startScanningImages()
	go perceptor.startPollingScanClient()
	go perceptor.startWritingPodUpdates()

	return &perceptor
}

func (perceptor *Perceptor) ImageScanStats() <-chan scanner.ImageScanStats {
	return perceptor.imageScanStats
}

func (perceptor *Perceptor) startPollingClusterManagerForNewPods() {
	for {
		select {
		case addPod := <-perceptor.clusterClient.PodAdd():
			perceptor.Cache.AddPod(addPod.New)
			// images := []string{}
			// for _, cont := range addPod.New.Spec.Containers {
			// 	images = append(images, cont.Image.Name()+", "+cont.Name)
			// }
			log.Infof("cluster manager event -- add pod: UID %s, name %s/%s", addPod.New.UID, addPod.New.Name, addPod.New.Namespace)
		case updatePod := <-perceptor.clusterClient.PodUpdate():
			log.Infof("cluster manager event -- update pod: UID %s, name %s/%s", updatePod.New.UID, updatePod.New.Name, updatePod.New.Namespace)
		case deletePod := <-perceptor.clusterClient.PodDelete():
			perceptor.Cache.DeletePod(deletePod)
			log.Infof("cluster manager event -- delete pod: ID %s", deletePod.ID)
		}
	}
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

func (perceptor *Perceptor) startWritingPodUpdates() {
	for {
		time.Sleep(20 * time.Second)
		log.Info("writing vulnerability cache into pod annotations")
		for podUID, pod := range perceptor.Cache.Pods {
			bdAnnotations, err := perceptor.clusterClient.GetBlackDuckPodAnnotations(pod.Namespace, pod.Name)
			if err != nil {
				log.Errorf("unable to get BlackDuckAnnotations for pod %s:%s -- %s", pod.Namespace, pod.Name, err.Error())
				continue
			}
			scanResults, err := perceptor.Cache.scanResults(podUID)
			if err != nil {
				log.Errorf("unable to retrieve scan results for Pod UID %s: %s", podUID, err.Error())
				continue
			}
			bdAnnotations.PolicyViolationCount = scanResults.PolicyViolationCount
			bdAnnotations.VulnerabilityCount = scanResults.VulnerabilityCount
			bdAnnotations.OverallStatus = scanResults.OverallStatus

			err = perceptor.clusterClient.SetBlackDuckPodAnnotations(pod.Namespace, pod.Name, *bdAnnotations)
			if err != nil {
				log.Errorf("unable to update BlackDuckAnnotations for pod %s:%s -- %s", pod.Namespace, pod.Name, err.Error())
				continue
			}
		}
	}
}
