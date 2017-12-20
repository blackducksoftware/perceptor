package scanner

import (
	"fmt"
	"os/exec"
	"os/user"

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
func NewHubScanClient(username string, password string, host string) (*HubScanClient, error) {
	baseURL := "https://" + host
	pf, err := NewProjectFetcher(username, password, baseURL)
	if err != nil {
		log.Errorf("unable to instantiate ProjectFetcher: %s", err.Error())
		return nil, err
	}
	// TODO this is a terrible way to locate the scan.docker.sh script;
	// need to figure out the right way to do this
	usr, err := user.Current()
	if err != nil {
		log.Errorf("unable to find current user's home dir: %s", err.Error())
		return nil, err
	}
	pathToScanner := usr.HomeDir + "/blackduck-bins/scan.cli-4.5.0-SNAPSHOT/bin/scan.docker.sh"

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

func (hsc *HubScanClient) FetchProject(projectName string) *Project {
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
		log.Debug("output from scan.docker.sh:\n%v\n", string(stdoutStderr))
		return err
	}
	log.Infof("successfully completed ./scan.docker.sh: %s", stdoutStderr)
	return nil
}
