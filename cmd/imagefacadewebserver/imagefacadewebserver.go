package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"

	pdocker "bitbucket.org/bdsengineering/perceptor/pkg/docker"
)

func main() {
	setupHTTPServer()
}

func setupHTTPServer() {
	imagePuller := pdocker.NewImagePuller()
	results := []pdocker.ImagePullStats{}
	http.HandleFunc("/pull", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Errorf("unable to read body for pod POST: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			var image common.Image
			err = json.Unmarshal(body, &image)
			if err != nil {
				log.Infof("unable to ummarshal JSON for pod POST: %s", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			go func() {
				results = append(results, imagePuller.PullImage(image))
			}()
		}
	})
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		statsBytes, err := json.Marshal(results)
		if err != nil {
			http.Error(w, err.Error(), 400)
		} else {
			fmt.Fprint(w, string(statsBytes))
		}
	})
	http.HandleFunc("/prettystats", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "start pretty stats:\n")
		for _, result := range results {
			fmt.Fprint(w, "stats: ")
			if result.Duration != nil {
				fmt.Fprintf(w, "seconds: %d", int(result.Duration.Seconds()))
			}
			if result.TarFileSizeMBs != nil {
				fmt.Fprintf(w, "  file size: %d", result.TarFileSizeMBs)
			}
			if result.Err != nil {
				fmt.Fprintf(w, "  error: %+v", result.Err)
			}
			fmt.Fprint(w, "\n")
		}
		fmt.Fprint(w, "end pretty stats")
	})

	log.Info("Serving")
	http.ListenAndServe(":3004", nil)
}
