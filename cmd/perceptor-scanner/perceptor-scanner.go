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

// Two things that should work:
// curl -X GET http://perceptor.bds-perceptor.svc.cluster.local:3001/metrics
// curl -X GET http://perceptor.bds-perceptor:3001/metrics
const (
	perceptorBaseURL     = "http://perceptor.bds-perceptor"
	nextImagePath        = "nextimage"
	finishedScanPath     = "finishedscan"
	perceptorPort        = "3001"
	perceptorScannerPort = "3003"
	perceptorProjectName = "Perceptor"
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
				job := scanner.NewScanJob(perceptorProjectName, *image)
				runScanJob(scanClient, *job)
			}
		}
	}()

	addr := fmt.Sprintf(":%s", perceptorScannerPort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

func requestScanJob() *common.Image {
	nextImageURL := fmt.Sprintf("%s:%s/%s", perceptorBaseURL, perceptorPort, nextImagePath)
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
		log.Infof("http POST request to %s succeeded", nextImageURL)
		return nextImage.Image
	}

	log.Errorf("http POST request to %s failed: %s", nextImageURL, err.Error())
	return nil
}

func runScanJob(scanClient *scanner.HubScanClient, job scanner.ScanJob) {
	scanResults, err := scanClient.Scan(job)
	finishedJob := api.FinishedScanClientJob{Err: err, Image: job.Image, Results: scanResults}
	finishScan(finishedJob)
}

func finishScan(results api.FinishedScanClientJob) {
	finishedScanURL := fmt.Sprintf("%s:%s/%s", perceptorBaseURL, perceptorPort, finishedScanPath)
	jsonBytes, err := json.Marshal(results)
	resp, err := http.Post(finishedScanURL, "application/json", bytes.NewBuffer(jsonBytes))
	defer resp.Body.Close()
	if resp.StatusCode == 200 && err == nil {
		log.Info("got successful response from %s", finishedScanURL)
	} else if err != nil {
		log.Errorf("unable to POST to %s: %s", finishedScanURL, err.Error())
	} else {
		log.Errorf("unable to POST to %s: got status code %d", finishedScanURL, resp.StatusCode)
	}
}
