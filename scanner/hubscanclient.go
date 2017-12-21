package scanner

import (
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// HubScanClient implements ScanClientInterface using
// the Black Duck hub and scan client programs.
type HubScanClient struct {
	host           string
	username       string
	password       string
	pathToScanner  string
	projectFetcher *ProjectFetcher
}

// NewHubScanClient requires login credentials in order to instantiate
// a HubScanClient.
func NewHubScanClient(username string, password string, host string, pathToScanner string) (*HubScanClient, error) {
	baseURL := "https://" + host
	pf, err := NewProjectFetcher(username, password, baseURL)
	if err != nil {
		log.Errorf("unable to instantiate ProjectFetcher: %s", err.Error())
		return nil, err
	}

	hsc := HubScanClient{
		host:           host,
		username:       username,
		password:       password,
		pathToScanner:  pathToScanner,
		projectFetcher: pf}
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
	return hsc.projectFetcher.FetchProjectOfName(projectName)
}

func (hsc *HubScanClient) Scan(job ScanJob) error {
	cmd := exec.Command(hsc.pathToScanner, // "./scan.docker.sh", // imagename, host, username "get", "pods", "-o", "json", "--all-namespaces")
		"--image",
		job.ImageName,
		"--project",
		job.ProjectName,
		"--host",
		hsc.host, // can switch to "localhost" for testing
		"--username",
		hsc.username) // can switch to "sysadmin" for testing
	fmt.Printf("running command %v for image %s\n", cmd, job.ImageName)
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
