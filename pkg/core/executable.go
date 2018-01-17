package core

import (
	"fmt"
	"os/exec"

	api "bitbucket.org/bdsengineering/perceptor/pkg/api"
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
	api.SetupHTTPServer(NewHttpResponder(perceptor))
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
	api.SetupHTTPServer(NewHttpResponder(perceptor))
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
