package core

import (
	"errors"
	"testing"

	"reflect"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/hub"
	"bitbucket.org/bdsengineering/perceptor/pkg/metrics"
	"bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	"github.com/prometheus/common/log"
)

func TestReducer(t *testing.T) {
	concurrentScanLimit := 1
	initialModel := NewModel(concurrentScanLimit)
	addPod := make(chan common.Pod)
	updatePod := make(chan common.Pod)
	deletePod := make(chan string)
	addImage := make(chan common.Image)
	updateAllPods := make(chan []common.Pod)
	postNextImage := make(chan func(image *common.Image))
	nextHubCheckImage := make(chan func(image *common.Image))
	finishScanClientJob := make(chan api.FinishedScanClientJob)
	hubScanResults := make(chan HubImageScan)
	reducer := newReducer(*initialModel,
		addPod,
		updatePod,
		deletePod,
		addImage,
		updateAllPods,
		postNextImage,
		finishScanClientJob,
		nextHubCheckImage,
		hubScanResults)

	image1 := *common.NewImage("image1", "fe67acf", "mfbd/image1")
	image2 := *common.NewImage("image2", "89ca3ec", "bds/image2")

	// 1. add a pod
	//   this should add all the images in the pod to the hub check queue (if they haven't already been added),
	//   add them to the image dictionary, and set their status to HubCheck
	go func() {
		addPod <- *common.NewPod("pod1", "uid1", "namespace1", []common.Container{
			*common.NewContainer(image1, "container1"),
			*common.NewContainer(image2, "container2"),
		})
	}()
	newModel := <-reducer.model
	if len(newModel.ImageHubCheckQueue) != 2 {
		t.Logf("expected there to be 2 images in queue, found %d", len(newModel.ImageHubCheckQueue))
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected there to be 0 images in queue, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults1, ok1 := newModel.Images[image1]
	if !ok1 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults1.ScanStatus != ScanStatusInHubCheckQueue {
		t.Logf("expected image1 ScanStatus to be InHubCheckQueue, but instead is %s", imageResults1.ScanStatus)
		t.Fail()
	}

	// 1a. move image1 from unknown into the hub check queue
	var nextCheckImage *common.Image
	go func() {
		nextHubCheckImage <- func(image *common.Image) {
			nextCheckImage = image
		}
	}()
	newModel = <-reducer.model
	if nextCheckImage == nil {
		t.Logf("expected to get an image for hub checking, got nothing")
		t.Fail()
	} else if *nextCheckImage != image1 {
		t.Logf("expected to get image1, got %s", nextCheckImage.HumanReadableName())
		t.Fail()
	}

	// 1b. move image1 from hub check queue into scan queue
	go func() {
		hubScanResults <- HubImageScan{
			Image: image1,
			Scan:  nil,
		}
	}()
	newModel = <-reducer.model

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
	} else if *nextImage != image1 {
		t.Logf("expected to get image1, got %s", nextImage.HumanReadableName())
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected there to be 0 images left in queue, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults2, ok2 := newModel.Images[image1]
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
	results3 := scanner.ScanClientJobResults{PullDuration: 32, ScanClientDuration: 17, TarFileSizeMBs: 22}
	log.Infof("is nil 1? %t", nextImage == nil)
	go func() {
		log.Infof("is nil 2? %t", nextImage == nil)
		finishScanClientJob <- api.FinishedScanClientJob{Err: nil, Image: *nextImage, Results: &results3}
	}()

	newModel = <-reducer.model
	imageScanStats3 := <-reducer.imageScanStats
	imageResults3, ok3 := newModel.Images[image1]
	if !ok3 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults3.ScanStatus != ScanStatusRunningHubScan {
		t.Logf("expected image1 ScanStatus to be RunningHubScan, but instead is %d", imageResults3.ScanStatus)
		t.Fail()
	}
	expected3 := metrics.ImageScanStats{PullDuration: 32, ScanDuration: 17, TarFileSizeMBs: 22}
	if imageScanStats3 != expected3 {
		t.Logf("expected image scan stats %v, but got %v", expected3, imageScanStats3)
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
		t.Logf("expected to not get an image, got %s", nextImage.HumanReadableName())
		t.Fail()
	}

	// 5. finish the hub scan for image1. this should:
	//    change the ScanStatus to complete
	//    add scan results
	go func() {
		hubScanResults <- HubImageScan{
			Image: image1,
			Scan: &hub.ImageScan{
				ScanSummary: hub.ScanSummary{Status: "COMPLETE"},
			},
		}
	}()
	newModel = <-reducer.model
	imageResults5, ok5 := newModel.Images[image1]
	if !ok5 {
		t.Logf("couldn't find image1 in image map")
		t.Fail()
	}
	if imageResults5.ScanStatus != ScanStatusComplete {
		t.Logf("expected image1 ScanStatus to be Complete, but instead is %s", imageResults5.ScanStatus)
		t.Fail()
	}
	expected5 := hub.ImageScan{}
	actual5 := *imageResults5.ScanResults

	// without using reflect, we get an error:
	//   invalid operation: expected5 != actual5 (struct containing hub.RiskProfile cannot be compared)
	if reflect.DeepEqual(expected5, actual5) {
		t.Logf("expected scan results to be %v, found %v", expected5, actual5)
		t.Fail()
	}

	// 6a. move image2 from unknown into the hub check queue
	go func() {
		nextHubCheckImage <- func(image *common.Image) {
			nextCheckImage = image
		}
	}()
	newModel = <-reducer.model
	if nextCheckImage == nil {
		t.Logf("expected to get an image for hub checking, got nothing")
		t.Fail()
	} else if *nextCheckImage != image2 {
		t.Logf("expected to get image2, got %s", nextCheckImage.HumanReadableName())
		t.Fail()
	}
	imageResults6a, ok6a := newModel.Images[image2]
	if !ok6a {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults6a.ScanStatus != ScanStatusCheckingHub {
		t.Logf("expected image2 ScanStatus to be CheckingHub, but instead is %d", imageResults6a.ScanStatus)
		t.Fail()
	}

	// 6b. move image2 from hub check queue into scan queue
	go func() {
		hubScanResults <- HubImageScan{
			Image: image2,
			Scan:  nil,
		}
	}()
	newModel = <-reducer.model
	imageResults6b, ok6b := newModel.Images[image2]
	if !ok6b {
		t.Logf("couldn't find image2 in image map")
		t.Fail()
	}
	if imageResults6b.ScanStatus != ScanStatusInQueue {
		t.Logf("expected image2 ScanStatus to be InQueue, but instead is %d", imageResults6b.ScanStatus)
		t.Fail()
	}

	// 6c. ask for the next image from the queue. this should:
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
	} else if *nextImage != image2 {
		t.Logf("expected to get image2, got %s", nextImage.HumanReadableName())
		t.Fail()
	}
	if len(newModel.ImageScanQueue) != 0 {
		t.Logf("expected the queue to be empty, found %d", len(newModel.ImageScanQueue))
		t.Fail()
	}
	imageResults6, ok6 := newModel.Images[image2]
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
	go func() {
		finishScanClientJob <- api.FinishedScanClientJob{Err: errors.New("oops"), Image: *nextImage, Results: nil}
	}()

	newModel = <-reducer.model
	// we don't get imageScanStats for scan client failures
	// imageScanStats7 := <-reducer.imageScanStats
	imageResults7, ok7 := newModel.Images[image2]
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
	if *nextImage != image2 {
		t.Logf("expected image name to be image2, got %s", nextImage.HumanReadableName())
	}
	imageResults8, ok8 := newModel.Images[image2]
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
		finishScanClientJob <- api.FinishedScanClientJob{Err: nil, Image: *nextImage, Results: &results9}
		log.Info("finished sending message 9")
	}()
	newModel = <-reducer.model
	imageScanStats9 := <-reducer.imageScanStats
	expected9 := metrics.ImageScanStats{PullDuration: 127, ScanDuration: 84, TarFileSizeMBs: 339}
	if imageScanStats9 != expected9 {
		t.Logf("expected image scan stats %v, but got %v", expected9, imageScanStats9)
		t.Fail()
	}

	// 10. finish hub scan with success
	go func() {
		hubScanResults <- HubImageScan{
			Image: image2,
			Scan: &hub.ImageScan{
				ScanSummary: hub.ScanSummary{Status: "Complete"},
			},
		}
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
