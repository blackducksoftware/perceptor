package scanner

import (
	"fmt"
	"os/exec"
	"os/user"
	"time"

	log "github.com/sirupsen/logrus"
)

// HubScanClient implements ScanClientInterface using
// the Black Duck hub and scan client programs.
type HubScanClient struct {
	host           string
	username       string
	password       string
	inProgressJobs map[string]ScanJob
	finishedJobs   map[string]ScanJob
	projectFetcher *ProjectFetcher
}

// NewHubScanClient requires login credentials in order to instantiate
// a HubScanClient.
func NewHubScanClient(username string, password string, host string) *HubScanClient {
	baseURL := "https://" + host
	pf, err := NewProjectFetcher(username, password, baseURL)
	if err != nil {
		panic("unable to instantiate ProjectFetcher: " + err.Error())
	}
	hsc := HubScanClient{
		host:           host,
		username:       username,
		password:       password,
		inProgressJobs: make(map[string]ScanJob),
		finishedJobs:   make(map[string]ScanJob),
		projectFetcher: pf}
	hsc.startPollingForFinishedScans()
	return &hsc
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

func (hsc *HubScanClient) startPollingForFinishedScans() {
	go func() {
		log.Info("starting to poll for finished scans")
		for {
			log.Infof("found %d finished jobs, %d in-progress jobs: %v <> %v", len(hsc.finishedJobs), hsc.finishedJobs, len(hsc.inProgressJobs), hsc.inProgressJobs)
			for _, name := range mapKeys(hsc.inProgressJobs) {
				project := hsc.projectFetcher.FetchProjectOfName(name)
				if project == nil {
					log.Infof("unable to fetch project %s, skipping", name)
					continue
				}
				log.Infof("fetched project of name %s: %v", name, project)
				scanJob, _ := hsc.inProgressJobs[name]
				log.Infof("removing project %s from in-progress jobs, adding to finished jobs", name)
				delete(hsc.inProgressJobs, name)
				hsc.finishedJobs[name] = scanJob
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func (hsc *HubScanClient) Scan(job ScanJob) {
	usr, _ := user.Current()
	pathToScanner := usr.HomeDir + "/blackduck-bins/scan.cli-4.5.0-SNAPSHOT/bin/scan.docker.sh"
	cmd := exec.Command(pathToScanner, // "./scan.docker.sh", // imagename, host, username "get", "pods", "-o", "json", "--all-namespaces")
		"--image",
		job.ImageName,
		"--project",
		job.ProjectName,
		"--host",
		hsc.host, // can switch to "localhost" for testing
		"--username",
		hsc.username) // can switch to "sysadmin" for testing
	fmt.Printf("running command %v for image %s\n", cmd, job.ImageName)
	hsc.inProgressJobs[job.ProjectName] = job
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		message := fmt.Sprintf("failed to run scan.docker.sh: %s", err.Error())
		log.Error(message)
		log.Debug("output from scan.docker.sh:\n%v\n", string(stdoutStderr))
	} else {
		log.Infof("successfully completed ./scan.docker.sh: %s", stdoutStderr)
	}
}

func (hsc *HubScanClient) GetFinishedJobs() []ScanJob {
	// TODO
	return []ScanJob{}
}
