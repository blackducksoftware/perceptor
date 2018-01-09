package main

import (
	"fmt"
	"net/http"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"

	pdocker "bitbucket.org/bdsengineering/perceptor/pkg/docker"
)

func main() {
	setupHTTPServer()
}

func setupHTTPServer() {
	http.HandleFunc("/pull", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		imageName := r.Form.Get("image")
		image := common.Image(imageName)
		err := pdocker.PullImage(image)
		if err != nil {
			http.Error(w, err.Error(), 400)
			log.Errorf("unable to pull image %s: %s", imageName, err.Error())
		} else {
			fmt.Fprintf(w, "successfully handled %s", imageName)
			log.Infof("successfully handled %s", imageName)
		}
	})
	log.Info("Serving")
	http.ListenAndServe(":3000", nil)
}
