package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func RunLocally(kubeconfigPath string) {
	log.Info("start")

	hubHost := "34.227.56.110.xip.io"
	// hubHost := "54.147.161.205.xip.io"
	clusterMasterURL := "https://" + hubHost + ":8443"

	openshiftMasterUsername := "admin"
	openshiftMasterPassword := "123"
	err := loginToOpenshift(clusterMasterURL, openshiftMasterUsername, openshiftMasterPassword)

	if err != nil {
		log.Errorf("unable to log in to openshift: %s", err.Error())
		panic(err)
	}

	log.Info("logged into openshift")

	hubUsername := "sysadmin"
	hubPassword := "blackduck"

	perceptor, err := NewPerceptor(clusterMasterURL, kubeconfigPath, hubUsername, hubPassword, hubHost)

	if err != nil {
		log.Errorf("unable to instantiate percepter: %s", err.Error())
		panic(err)
	}

	log.Info("instantiated perceptor: %v", perceptor)

	log.Info("finished starting")
	setupHTTPServer(perceptor)
}

func RunFromInsideCluster() {
	log.Info("start")

	hubHost := "34.227.56.110.xip.io"
	// hubHost := "54.147.161.205.xip.io"

	hubUsername := "sysadmin"
	hubPassword := "blackduck"

	perceptor, err := NewPerceptorFromCluster(hubUsername, hubPassword, hubHost)

	if err != nil {
		log.Errorf("unable to instantiate percepter: %s", err.Error())
		panic(err)
	}

	log.Info("instantiated perceptor: %v", perceptor)

	log.Info("finished starting")
	setupHTTPServer(perceptor)
}

func loginToOpenshift(host string, username string, password string) error {
	// TODO do we need to `oc logout` first?
	cmd := exec.Command("oc", "login", host, "--insecure-skip-tls-verify=true", "-u", username, "-p", password)
	fmt.Println("running command 'oc login ...'")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("unable to login to oc: %s, %s", stdoutStderr, err)
	}
	log.Infof("finished `oc login`: %s", stdoutStderr)
	return err
}

// other stuff

func setupHTTPServer(perceptor *Perceptor) {
	http.Handle("/metrics", metricsHandler(perceptor.ImageScanStats()))
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Info("shutting down")
		fmt.Fprint(w, "Shutdown now!\n")
		os.Exit(0)
	})
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		jsonBytes, err := json.Marshal(perceptor)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to serialize model: %s", err.Error()), 500)
			return
		}
		jsonString := string(jsonBytes)
		fmt.Fprint(w, jsonString)
	})
	log.Info("Serving")
	// TODO response to an edit-pods endpoint or something

	// TODO make this configurable - maybe even viperize it.
	http.ListenAndServe(":3000", nil)
	log.Info("Http server started!")
}
