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
	host       string
	username   string
	password   string
	hubFetcher *HubFetcher
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
		host:       host,
		username:   username,
		password:   password,
		hubFetcher: hf}
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
	err := pdocker.PullImage(job.Image)
	if err != nil {
		log.Errorf("unable to pull docker image %s: %s", job.Image.Name, err.Error())
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
	/*
		cmd := exec.Command(hsc.pathToScanner,
			"--image",
			job.ImageName,
			"--project",
			job.ProjectName,
			"--host",
			hsc.host,
			"--username",
			hsc.username)
	*/
	log.Infof("running command %v for image %s\n", cmd, job.Image.Name)
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

/*
func (hsc *HubScanClient) ScanDockerSh(job ScanJob) error {
	cmd := exec.Command(hsc.pathToScanner, // "./scan.docker.sh",
		"--image",
		job.ImageName,
		"--project",
		job.ProjectName,
		"--host",
		hsc.host,
		"--username",
		hsc.username)
	log.Infof("running command %v for image %s\n", cmd, job.ImageName)
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
*/
