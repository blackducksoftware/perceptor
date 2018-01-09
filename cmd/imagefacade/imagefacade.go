// Executable that caches images in a directory as tarballs.
package main

import (
	"flag"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"

	pdocker "bitbucket.org/bdsengineering/perceptor/pkg/docker"
)

type input struct {
	fromImage string
	tag       string
	digest    string // need this so we know what to look up inthe api.
}

var in input

func init() {
	// go run cmd/imagefacade/imagefacade.go -fromImage registry.kipp.blackducksoftware.com/blackducksoftware/hub-jobrunner:4.5.0
	flag.StringVar(&in.fromImage, "fromImage", "", "imageDigest or name Will have .tar at the end.")
	flag.StringVar(&in.tag, "tag", "", "tag, empty is ok.")
}

func main() {
	flag.Parse()

	if in.fromImage == "" {
		panic("Need -fromImage <image>")
	}

	image := common.Image(in.fromImage)
	err := pdocker.PullImage(image)

	if err != nil {
		log.Errorf("Error while making tar file: %s", err)
	} else {
		log.Infof("Ready to scan !!!!! %s %s", in.fromImage, in.tag)
	}
}
