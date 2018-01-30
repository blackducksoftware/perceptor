package core

import (
	"fmt"
	"net/http"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func RunLocally(kubeconfigPath string) {
	log.Info("start")

	config := &PerceptorConfig{
		HubHost: "34.227.56.110.xip.io",
		HubUser: "sysadmin",
		HubUserPassword: "blackduck",
	}
	clusterMasterURL := "https://" + config.HubHost + ":8443"

	openshiftMasterUsername := "admin"
	openshiftMasterPassword := "123"
	err := loginToOpenshift(clusterMasterURL, openshiftMasterUsername, openshiftMasterPassword)

	if err != nil {
		log.Errorf("unable to log in to openshift: %s", err.Error())
		panic(err)
	}

	log.Info("logged into openshift")

	perceptor, err := NewPerceptor(config)

	if err != nil {
		log.Errorf("unable to instantiate percepter: %s", err.Error())
		panic(err)
	}

	log.Info("instantiated perceptor: %v", perceptor)

	http.ListenAndServe(":3000", nil)
	log.Info("Http server started!")
}

func RunFromInsideCluster() {
	log.Info("start")

	config, err := GetPerceptorConfig()
	if err != nil {
		log.Error("Failed to load configuration: %s", err.Error())
		panic(err)
	}

	// TODO: Start watching the config file.  Will need to refactor to allow hub client to be
	// recreated, possibly other things

	perceptor, err := NewPerceptor(config)

	if err != nil {
		log.Errorf("unable to instantiate percepter: %s", err.Error())
		panic(err)
	}

	log.Info("instantiated perceptor: %v", perceptor)

	// TODO make this configurable - maybe even viperize it.
	http.ListenAndServe(":3000", nil)
	log.Info("Http server started!")
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
