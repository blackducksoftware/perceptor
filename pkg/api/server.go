package httpserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

func SetupHTTPServer(responder Responder) {
	// state of the program
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			responder.GetMetrics(w, r)
		} else {
			http.Error(w, "404 not found.", http.StatusNotFound)
		}
	})
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			responder.GetModel(w, r)
		} else {
			http.Error(w, "404 not found.", http.StatusNotFound)
		}
	})

	// for receiving data from perceiver
	http.HandleFunc("/pod", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
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
			http.Error(w, "404 not found", http.StatusNotFound)
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
			http.Error(w, "", http.StatusNotFound)
		}
	})

	// TODO make this configurable - maybe even viperize it.
	http.ListenAndServe(":3000", nil)
	log.Info("Http server started!")
}
