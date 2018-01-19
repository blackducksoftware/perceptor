package core

import (
	"errors"
	"testing"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	"github.com/prometheus/common/log"
)

func TestReducer(t *testing.T) {
	initialModel := NewModel()
	addPod := make(chan common.Pod)
	updatePod := make(chan common.Pod)
	deletePod := make(chan string)
	postNextImage := make(chan func(image *common.Image))
	finishScanClientJob := make(chan FinishedScanClientJob)
	hubScanResults := make(chan scanner.Project)
	reducer := newReducer(*initialModel, addPod, updatePod, deletePod, postNextImage, finishScanClientJob, hubScanResults)

	// keep imageScanStats from blocking us
	go func() {
		for {
			select {
			case <-reducer.imageScanStats:
				break
			}
		}
	}()

	// 1. add a pod
	//   this should add all the images in the pod to the scan queue (if they haven't already been added)
	//   add them to the image dictionary, and set their status to InQueue
	go func() {
		addPod <- *common.NewPod("pod1", "uid1", "namespace1", []common.Container{
			*common.NewContainer(common.Image("image1"), "container1"),
			*common.NewContainer(common.Image("image2"), "container2"),
		})
	}()
	newModel := <-reducer.model
	if len(newModel.ImageScanQueue) != 2 {
		t.Logf("expected there to be 2 images in queue, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults1, ok1 := newModel.Images["image1"]
	if !ok1 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults1.ScanStatus != ScanStatusInQueue {
		t.Logf("expected image1 ScanStatus to be InQueue, but instead is %d", imageResults1.ScanStatus)
		t.Fail()
	}

	// 2. ask for the next image from the queue. this should:
	//   remove the first item from the queue
	//   change its status to InProgress
	var nextImage *common.Image
	go func() {
		postNextImage <- func(image *common.Image) {
			nextImage = image
		}
	}()

	newModel = <-reducer.model
	if nextImage == nil {
		t.Logf("expected to get an image, got nothing")
		t.Fail()
	} else if *nextImage != common.Image("image1") {
		t.Logf("expected to get image1, got %s", nextImage.Name())
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 1 {
		t.Logf("expected there to only be 1 image left in queue, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults2, ok2 := newModel.Images["image1"]
	if !ok2 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults2.ScanStatus != ScanStatusRunningScanClient {
		t.Logf("expected image1 ScanStatus to be RunningScanClient, but instead is %d", imageResults2.ScanStatus)
		t.Fail()
	}

	// 3. finish a scan
	//   this should cause the image status to be set to running hub scan,
	//   and results to be added in the image dict
	results := scanner.ScanClientJobResults{PullDuration: 32, ScanClientDuration: 17, TarFileSizeMBs: 22}
	go func() {
		finishScanClientJob <- FinishedScanClientJob{err: nil, image: *nextImage, results: &results}
	}()

	newModel = <-reducer.model
	imageResults3, ok3 := newModel.Images["image1"]
	if !ok3 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults3.ScanStatus != ScanStatusRunningHubScan {
		t.Logf("expected image1 ScanStatus to be RunningHubScan, but instead is %d", imageResults3.ScanStatus)
		t.Fail()
	}

	// 4. ask for the next image from the queue. this hits the concurrency limit,
	//    so it should not do anything
	go func() {
		postNextImage <- func(image *common.Image) {
			nextImage = image
		}
	}()
	newModel = <-reducer.model
	if nextImage != nil {
		t.Logf("expected to not get an image, got %s", nextImage.Name())
		t.Fail()
	}

	// 5. finish the hub scan for image1. this should:
	//    change the ScanStatus to complete
	//    add scan results
	go func() {
		hubScanResults <- scanner.Project{Name: "Perceptor", Source: "", Versions: []scanner.Version{
			scanner.Version{VersionName: "image1", CodeLocations: []scanner.CodeLocation{
				scanner.CodeLocation{ScanSummaries: []scanner.ScanSummary{
					scanner.ScanSummary{Status: "COMPLETE"},
				}},
			}},
		}}
	}()
	newModel = <-reducer.model
	imageResults5, ok5 := newModel.Images["image1"]
	if !ok5 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults5.ScanStatus != ScanStatusComplete {
		t.Logf("expected image1 ScanStatus to be Complete, but instead is %d", imageResults5.ScanStatus)
		t.Fail()
	}
	expected5 := ScanResults{OverallStatus: "", PolicyViolationCount: 0, VulnerabilityCount: 0}
	actual5 := *imageResults5.ScanResults
	if expected5 != actual5 {
		t.Logf("expected scan results to be %v, found %v", expected5, actual5)
		t.Fail()
	}

	// 6. ask for the next image from the queue. this should:
	//   remove the first item from the queue
	//   change its status to InProgress
	go func() {
		postNextImage <- func(image *common.Image) {
			nextImage = image
		}
	}()
	newModel = <-reducer.model
	if nextImage == nil {
		t.Logf("expected to get an image, got nothing")
		t.Fail()
	} else if *nextImage != common.Image("image2") {
		t.Logf("expected to get image2, got %s", nextImage.Name())
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected the queue to be empty, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults6, ok6 := newModel.Images["image2"]
	if !ok6 {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults6.ScanStatus != ScanStatusRunningScanClient {
		t.Logf("expected image2 ScanStatus to be RunningScanClient, but instead is %d", imageResults6.ScanStatus)
		t.Fail()
	}

	// 7. finish a scan with an error
	//   this should cause the image to get put back in the queue,
	//   and the status set back to InQueue
	results7 := scanner.ScanClientJobResults{PullDuration: 32, ScanClientDuration: 17, TarFileSizeMBs: 22}
	finishScanClientJob <- FinishedScanClientJob{err: errors.New("oops"), image: *nextImage, results: &results7}

	newModel = <-reducer.model
	imageResults7, ok7 := newModel.Images["image2"]
	if !ok7 {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults7.ScanStatus != ScanStatusInQueue {
		t.Logf("expected image7 ScanStatus to be InQueue, but instead is %d", imageResults7.ScanStatus)
		t.Fail()
	}

	// 8. ask for next image, get image2 again
	log.Info("about to run gofunc for message 8")
	go func() {
		log.Info("send message 8")
		postNextImage <- func(image *common.Image) {
			nextImage = image
		}
		log.Info("finished sending message 8")
	}()
	log.Info("get model 8")
	newModel = <-reducer.model
	log.Info("finished getting model 8")
	if nextImage == nil {
		t.Logf("expected to get an image, got nothing")
		t.Fail()
	}
	if nextImage.Name() != "image2" {
		t.Logf("expected image name to be image2, got %s", nextImage.Name())
	}
	imageResults8, ok8 := newModel.Images["image2"]
	log.Info("check results 8")
	if !ok8 {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults8.ScanStatus != ScanStatusRunningScanClient {
		t.Logf("expected image7 ScanStatus to be InQueue, but instead is %d", imageResults8.ScanStatus)
		t.Fail()
	}

	// 9. finish scan client with success
	results9 := scanner.ScanClientJobResults{PullDuration: 127, ScanClientDuration: 84, TarFileSizeMBs: 339}
	log.Info("about to run gofunc for message 9")
	go func() {
		log.Info("send message 9")
		finishScanClientJob <- FinishedScanClientJob{err: nil, image: *nextImage, results: &results9}
		log.Info("finished sending message 9")
	}()
	newModel = <-reducer.model

	// 10. finish hub scan with success
	go func() {
		hubScanResults <- scanner.Project{Name: "Perceptor", Source: "", Versions: []scanner.Version{
			scanner.Version{VersionName: "image2", CodeLocations: []scanner.CodeLocation{
				scanner.CodeLocation{ScanSummaries: []scanner.ScanSummary{
					scanner.ScanSummary{Status: "COMPLETE"}}}}}}}
	}()
	newModel = <-reducer.model

	// 11. ask for next image, get nil because queue is empty
	go func() {
		postNextImage <- func(image *common.Image) {
			log.Infof("image: %v, %t", image, image == nil)
			nextImage = image
		}
	}()
	newModel = <-reducer.model
	if nextImage != nil {
		t.Logf("expected to get nothing, got %v", nextImage)
		t.Fail()
	}

	log.Info("done with all messages")
}
