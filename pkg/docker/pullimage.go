package docker

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

const (
	dockerSocketPath = "/var/run/docker.sock"
)

type ImagePuller struct {
	rootTarballDir string
	client         *http.Client
}

func NewImagePuller() *ImagePuller {
	fd := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", dockerSocketPath)
	}
	tr := &http.Transport{Dial: fd}
	client := &http.Client{Transport: tr}
	return &ImagePuller{rootTarballDir: "./tmp", client: client}
}

// PullImage gives us access to a docker image by:
//   1. hitting a docker create endpoint (?)
//   2. pulling down the newly created image and saving as a tarball
// It does this by accessing the host's docker daemon, locally, over the docker
// socket.  This gives us a window into any images that are local.
func (ip *ImagePuller) PullImage(image common.Image) ImagePullStats {
	stats := ImagePullStats{}
	start := time.Now()

	err := ip.createImageInLocalDocker(image)
	if err != nil {
		stats.Err = &ImagePullError{Code: ErrorTypeUnableToCreateImage, RootCause: err}
		return stats
	}

	log.Infof("Processing image: %s", image.HumanReadableName())

	fileSize, pullError := ip.saveImageToTar(image)
	if err != nil {
		stats.Err = pullError
		return stats
	}

	stop := time.Now()

	log.Infof("Ready to scan image %s at path %s", image.HumanReadableName(), image.TarFilePath())
	duration := stop.Sub(start)
	stats.Duration = &duration
	stats.TarFileSizeMBs = fileSize
	return stats
}

// createImageInLocalDocker could also be implemented using curl:
// this example hits ... ? the default registry?  docker hub?
//   curl --unix-socket /var/run/docker.sock -X POST http://localhost/images/create?fromImage=alpine
// this example hits the kipp registry:
//   curl --unix-socket /var/run/docker.sock -X POST http://localhost/images/create\?fromImage\=registry.kipp.blackducksoftware.com%2Fblackducksoftware%2Fhub-jobrunner%3A4.5.0
//
func (ip *ImagePuller) createImageInLocalDocker(image common.Image) (err error) {
	imageURL := image.CreateURL()
	log.Infof("Attempting to create %s ......", imageURL)
	resp, err := ip.client.Post(imageURL, "", nil)
	defer resp.Body.Close()

	if resp.StatusCode == 200 && err == nil {
		log.Infof("Create succeeded for %s %v", imageURL, resp)
	} else if err == nil {
		// This should get hit if there's a 404
		log.Infof("Create may have failed for %s: status code %d, response", imageURL, resp.StatusCode, resp)
	} else {
		log.Errorf("Create failed for image %s: %s", imageURL, err.Error())
	}
	return err
}

// saveImageToTar: part of what it does is to issue an http request similar to the following:
//   curl --unix-socket /var/run/docker.sock -X GET http://localhost/images/openshift%2Forigin-docker-registry%3Av3.6.1/get
func (ip *ImagePuller) saveImageToTar(image common.Image) (*int, *ImagePullError) {
	log.Infof("Making http request: [%s]", image.GetURL())
	resp, err := ip.client.Get(image.GetURL())
	if err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToGetImage, RootCause: err}
	} else if resp.StatusCode != http.StatusOK {
		return nil, &ImagePullError{
			Code:      ErrorTypeBadStatusCodeFromGetImage,
			RootCause: fmt.Errorf("HTTP ERROR: received status != 200 from %s: %s", image.GetURL(), resp.Status)}
	}

	log.Infof("GET request for %s successful", image.GetURL())

	body := resp.Body
	defer func() {
		body.Close()
	}()
	log.Info("Starting to write file contents to a tar file.")
	tarFilePath := image.TarFilePath()
	log.Infof("Tar File Path: %s", tarFilePath)

	// just need to create `./tmp` if it doesn't already exist
	os.Mkdir(ip.rootTarballDir, 0755)

	f, err := os.OpenFile(tarFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToCreateTarFile, RootCause: err}
	}
	if _, err = io.Copy(f, body); err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToCopyTarFile, RootCause: err}
	}

	// What's the right way to get the size of the file?
	//  1. resp.ContentLength
	//  2. check the size of the file after it's written
	// fileSizeInMBs := int(resp.ContentLength / (1024 * 1024))
	stats, err := os.Stat(tarFilePath)

	if err != nil {
		return nil, &ImagePullError{Code: ErrorTypeUnableToGetFileStats, RootCause: err}
	}

	fileSizeInMBs := int(stats.Size() / (1024 * 1024))

	return &fileSizeInMBs, nil
}
