package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/scanner"
	log "github.com/sirupsen/logrus"
)

// TODO metrics
// number of images scanned
// file size
// pull duration
// get duration
// scan client duration
// number of successes
// number of failures
// amount of time (or cycles?) idled
// number of times asked for a job and didn't get one

func main() {
	log.Info("started")

	// TODO viperize
	username := "sysadmin"
	password := "blackduck"
	host := "34.227.56.110.xip.io"

	scanClient, err := scanner.NewHubScanClient(username, password, host)
	if err != nil {
		log.Errorf("unable to instantiate hub scan client: %s", err.Error())
		panic(err)
	}

	go func() {
		for {
			time.Sleep(20 * time.Second)
			image := requestScanJob()
			if image != nil {
				job := scanner.NewScanJob(*image)
				runScanJob(scanClient, *job)
			}
		}
	}()

	addr := fmt.Sprintf(":%s", api.PerceptorScannerPort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

func requestScanJob() *common.Image {
	nextImageURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.NextImagePath)
	resp, err := http.Post(nextImageURL, "", bytes.NewBuffer([]byte{}))
	if err != nil {
		log.Errorf("unable to POST to %s: %s", nextImageURL, err.Error())
		return nil
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("unable to read resp body from %s: %s", nextImageURL, err.Error())
		return nil
	}

	var nextImage api.NextImage
	err = json.Unmarshal(bodyBytes, &nextImage)
	if err == nil && resp.StatusCode == 200 {
		imageName := "null"
		if nextImage.Image != nil {
			imageName = nextImage.Image.ShaName()
		}
		log.Infof("http POST request to %s succeeded, got image %s", nextImageURL, imageName)
		return nextImage.Image
	}

	log.Errorf("http POST request to %s failed: %s", nextImageURL, err.Error())
	return nil
}

func runScanJob(scanClient *scanner.HubScanClient, job scanner.ScanJob) {
	scanResults, err := scanClient.Scan(job)
	errorString := ""
	if err != nil {
		errorString = err.Error()
	}
	finishedJob := api.FinishedScanClientJob{Err: errorString, Image: job.Image, Results: scanResults}
	log.Infof("about to finish job, going to send over %v", finishedJob)
	finishScan(finishedJob)
}

func finishScan(results api.FinishedScanClientJob) {
	finishedScanURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.FinishedScanPath)
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		log.Errorf("unable to marshal json for finished job: %s", err.Error())
		panic(err)
	}
	log.Infof("about to send over json text for finishing a job: %s", string(jsonBytes))
	resp, err := http.Post(finishedScanURL, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Errorf("unable to POST to %s: %s", finishedScanURL, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		log.Infof("POST to %s succeeded", finishedScanURL)
	} else {
		log.Errorf("POST to %s failed with status code %d", finishedScanURL, resp.StatusCode)
	}
}
