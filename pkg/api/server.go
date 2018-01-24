package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

func SetupHTTPServer(responder Responder) {
	// state of the program
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			responder.GetMetrics(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			fmt.Fprint(w, responder.GetModel())
		} else {
			http.NotFound(w, r)
		}
	})

	// for receiving data from perceiver
	http.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Errorf("unable to read body for pod POST: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var pod common.Pod
			err = json.Unmarshal(body, &pod)
			if err != nil {
				log.Infof("unable to ummarshal JSON for pod POST: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			responder.AddPod(pod)
			fmt.Fprint(w, "")
		case "PUT":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			var pod common.Pod
			err = json.Unmarshal(body, &pod)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			responder.UpdatePod(pod)
			fmt.Fprint(w, "")
		case "DELETE":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			responder.DeletePod(string(body))
			fmt.Fprint(w, "")
		default:
			http.NotFound(w, r)
		}
	})
	// http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
	// 	body, err := ioutil.ReadAll(r.Body)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), 400)
	// 		return
	// 	}
	// 	var image common.Image
	// 	err = json.Unmarshal(body, &image)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), 400)
	// 		return
	// 	}
	// 	responder.Image(w, r, image)
	// })

	// for providing data to perceiver
	http.HandleFunc("/scanresults", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			scanResults := responder.GetScanResults()
			jsonBytes, err := json.Marshal(scanResults)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			fmt.Fprint(w, string(jsonBytes))
		} else {
			http.NotFound(w, r)
		}
	})

	// for providing data to scanners
	http.HandleFunc("/nextimage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var wg sync.WaitGroup
			wg.Add(1)
			responder.GetNextImage(func(nextImage NextImage) {
				jsonBytes, err := json.Marshal(nextImage)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				fmt.Fprint(w, string(jsonBytes))
				wg.Done()
			})
			wg.Wait()
		} else {
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/finishedscan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			var scanResults FinishedScanClientJob
			err = json.Unmarshal(body, &scanResults)
			responder.PostFinishScan(scanResults)
			fmt.Fprint(w, "")
		} else {
			http.NotFound(w, r)
		}
	})
}
