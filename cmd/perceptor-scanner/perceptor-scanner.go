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
	"github.com/spf13/viper"
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

	config, err := GetScannerConfig()
	if err != nil {
		log.Error("Failed to load configuration")
		panic(err)
	}

	scanClient, err := scanner.NewHubScanClient(config.HubHost, config.HubUser, config.HubUserPassword)
	if err != nil {
		log.Errorf("unable to instantiate hub scan client: %s", err.Error())
		panic(err)
	}

	imageScanStats := make(chan scanner.ScanClientJobResults)
	httpStats := make(chan scanner.HttpResult)

	go func() {
		for {
			time.Sleep(20 * time.Second)
			err := requestAndRunScanJob(scanClient, imageScanStats, httpStats)
			if err != nil {
				log.Errorf("error requesting or running scan job: %v", err)
			}
		}
	}()

	http.Handle("/metrics", scanner.ScannerMetricsHandler(imageScanStats, httpStats))

	addr := fmt.Sprintf(":%s", api.PerceptorScannerPort)
	http.ListenAndServe(addr, nil)
	log.Info("Http server started!")
}

func requestAndRunScanJob(scanClient *scanner.HubScanClient, imageScanStats chan<- scanner.ScanClientJobResults, httpStats chan<- scanner.HttpResult) error {
	image := requestScanJob(httpStats)
	if image == nil {
		return nil
	}
	job := scanner.NewScanJob(*image)
	scanResults := scanClient.Scan(*job)
	imageScanStats <- scanResults
	errorString := ""
	if scanResults.Err != nil {
		errorString = scanResults.Err.Error()
	}
	finishedJob := api.FinishedScanClientJob{Err: errorString, Image: job.Image}
	log.Infof("about to finish job, going to send over %v", finishedJob)
	return finishScan(finishedJob, httpStats)
}

func requestScanJob(httpStats chan<- scanner.HttpResult) *common.Image {
	nextImageURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.NextImagePath)
	resp, err := http.Post(nextImageURL, "", bytes.NewBuffer([]byte{}))
	httpStats <- scanner.HttpResult{Path: scanner.PathGetNextImage, StatusCode: resp.StatusCode}
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

func finishScan(results api.FinishedScanClientJob, httpStats chan<- scanner.HttpResult) error {
	finishedScanURL := fmt.Sprintf("%s:%s/%s", api.PerceptorBaseURL, api.PerceptorPort, api.FinishedScanPath)
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		log.Errorf("unable to marshal json for finished job: %s", err.Error())
		return err
	}
	log.Infof("about to send over json text for finishing a job: %s", string(jsonBytes))
	// TODO change to exponential backoff or something ... but don't loop indefinitely in production
	for {
		resp, err := http.Post(finishedScanURL, "application/json", bytes.NewBuffer(jsonBytes))
		httpStats <- scanner.HttpResult{Path: scanner.PathPostScanResults, StatusCode: resp.StatusCode}
		if err != nil {
			log.Errorf("unable to POST to %s: %s", finishedScanURL, err.Error())
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Errorf("POST to %s failed with status code %d", finishedScanURL, resp.StatusCode)
			continue
		}

		log.Infof("POST to %s succeeded", finishedScanURL)
		return nil
	}
}

// ScannerConfig contains all configuration for Perceptor
type ScannerConfig struct {
	HubHost         string
	HubUser         string
	HubUserPassword string
}

// GetScannerConfig returns a configuration object to configure Perceptor
func GetScannerConfig() (*ScannerConfig, error) {
	var cfg *ScannerConfig

	viper.SetConfigName("scanner_conf")
	viper.AddConfigPath("/etc/scanner_conf")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return cfg, nil
}
