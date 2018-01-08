package common

import (
	"fmt"
	"net/url"
	"strings"
)

type Image struct {
	Name string
}

func NewImage(name string) *Image {
	return &Image{Name: name}
}

func (image *Image) TarFilePath() string {
	// have to get rid of `/` so that it's not interpreted as directory separators
	sanitizedName := strings.Replace(image.Name, "/", "_", -1)
	// TODO use os.join or something
	return fmt.Sprintf("./tmp/%s.tar", sanitizedName)
}

func (image *Image) URLEncodedName() string {
	return url.QueryEscape(image.Name)
}

func (image *Image) CreateURL() string {
	// TODO v1.24 refers to the docker version.  figure out how to avoid hard-coding this
	// TODO can probably use the docker api code for this
	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s", image.URLEncodedName())
	//	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s&tag=%s", image.name, image.tag)
}

func (image *Image) GetURL() string {
	// TODO we'll leave off user for now, but maybe it should be added back in later ???
	//   the digest could also be added in
	// imageName := fmt.Sprintf("%s%s%s%s%s", image.user, "%2F", image.name, "%3A", image.tag)
	// TODO let's maybe trying keeping everything together in image -- example of which is:
	//   172.30.89.171:5000/blackduck-scan/hub_ose_arbiter:4.3.0
	// imageName := fmt.Sprintf("%s%s%s", image.name, "%3A", image.tag)
	return fmt.Sprintf("/images/%s/get", image.URLEncodedName())
}
