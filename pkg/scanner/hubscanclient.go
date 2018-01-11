package scanner

import (
	"fmt"
	"os/exec"

	pdocker "bitbucket.org/bdsengineering/perceptor/pkg/docker"
	log "github.com/sirupsen/logrus"
)

// HubScanClient implements ScanClientInterface using
// the Black Duck hub and scan client programs.
type HubScanClient struct {
	host        string
	username    string
	password    string
	hubFetcher  *HubFetcher
	imagePuller *pdocker.ImagePuller
}

// NewHubScanClient requires login credentials in order to instantiate
// a HubScanClient.
func NewHubScanClient(username string, password string, host string) (*HubScanClient, error) {
	baseURL := "https://" + host
	hf, err := NewHubFetcher(username, password, baseURL)
	if err != nil {
		log.Errorf("unable to instantiate ProjectFetcher: %s", err.Error())
		return nil, err
	}

	hsc := HubScanClient{
		host:        host,
		username:    username,
		password:    password,
		hubFetcher:  hf,
		imagePuller: pdocker.NewImagePuller()}
	return &hsc, nil
}

func mapKeys(m map[string]ScanJob) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func (hsc *HubScanClient) FetchProject(projectName string) (*Project, error) {
	return hsc.hubFetcher.FetchProjectOfName(projectName)
}

func (hsc *HubScanClient) Scan(job ScanJob) error {
	err := hsc.imagePuller.PullImage(job.Image)
	if err != nil {
		log.Errorf("unable to pull docker image %s: %s", job.Image.Name(), err.Error())
		return err
	}
	// TODO coupla problems here:
	//   1. hardcoded path
	//   2. hardcoded version number
	scanCliImplJarPath := "./dependencies/scan.cli-4.3.0/lib/cache/scan.cli.impl-standalone.jar"
	scanCliJarPath := "./dependencies/scan.cli-4.3.0/lib/scan.cli-4.3.0-standalone.jar"
	path := job.Image.TarFilePath()
	cmd := exec.Command("java",
		"-Xms512m",
		"-Xmx4096m",
		"-Dblackduck.scan.cli.benice=true",
		"-Dblackduck.scan.skipUpdate=true",
		"-Done-jar.silent=true",
		"-Done-jar.jar.path="+scanCliImplJarPath,
		"-jar", scanCliJarPath,
		"--host", hsc.host,
		"--port", "443", // "--port", "8443", // TODO or should this be 8080 or something else? or should we just leave it off and let it default?
		"--scheme", "https", // TODO or should this be http?
		"--project", job.ProjectName,
		"--release", job.Image.Name(),
		"--username", hsc.username,
		"--name", job.Image.Name(), // this is the scan name, maybe ... ???
		"--insecure", // TODO not sure about this
		"-v",
		path)
	log.Infof("running command %v for image %s\n", cmd, job.Image.Name)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		message := fmt.Sprintf("failed to run java scanner: %s", err.Error())
		log.Error(message)
		log.Errorf("output from java scanner:\n%v\n", string(stdoutStderr))
		return err
	}
	log.Infof("successfully completed java scanner: %s", stdoutStderr)
	return nil
}

func (hsc *HubScanClient) ScanCliSh(job ScanJob) error {
	pathToScanner := "./dependencies/scan.cli-4.3.0/bin/scan.cli.sh"
	cmd := exec.Command(pathToScanner,
		"--project", job.ProjectName,
		"--host", hsc.host,
		"--port", "443",
		"--insecure",
		"--username", hsc.username,
		job.Image.Name())
	log.Infof("running command %v for image %s\n", cmd, job.Image.Name())
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		message := fmt.Sprintf("failed to run scan.cli.sh: %s", err.Error())
		log.Error(message)
		log.Errorf("output from scan.cli.sh:\n%v\n", string(stdoutStderr))
		return err
	}
	log.Infof("successfully completed scan.cli.sh: %s", stdoutStderr)
	return nil
}

func (hsc *HubScanClient) ScanDockerSh(job ScanJob) error {
	pathToScanner := "./dependencies/scan.cli-4.3.0/bin/scan.docker.sh"
	cmd := exec.Command(pathToScanner,
		"--image", job.Image.Name(),
		"--name", job.Image.Name(),
		"--release", job.Image.Name(),
		"--project", job.ProjectName,
		"--host", hsc.host,
		"--username", hsc.username)
	log.Infof("running command %v for image %s\n", cmd, job.Image.Name())
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		message := fmt.Sprintf("failed to run scan.docker.sh: %s", err.Error())
		log.Error(message)
		log.Errorf("output from scan.docker.sh:\n%v\n", string(stdoutStderr))
		return err
	}
	log.Infof("successfully completed ./scan.docker.sh: %s", stdoutStderr)
	return nil
}
