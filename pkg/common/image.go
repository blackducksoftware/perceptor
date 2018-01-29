package common

import (
	"fmt"
	"net/url"
)

type Image struct {
	// Name combines Host, User, and Project
	// DockerImage is the kubernetes .Image string, which may or may not include the registry, user, tag, and sha
	//   DockerImage should probably only be used as a human-readable string, not for storing or organizing
	//   data, because it is so nebulous and ambiguous.
	Name        string
	Sha         string
	DockerImage string
}

func NewImage(name string, sha string, dockerImage string) *Image {
	return &Image{Name: name, Sha: sha, DockerImage: dockerImage}
}

func (image *Image) HubProjectName() string {
	return image.Name
}

func (image *Image) HubVersionName() string {
	return image.Sha
}

func (image *Image) HubScanName() string {
	return image.DockerImage
}

// Name returns a nice, easy to read string
func (image *Image) HumanReadableName() string {
	return image.DockerImage
}

// FullName combines Name with the image sha
func (image *Image) ShaName() string {
	return fmt.Sprintf("%s/@sha256:%s", image.Name, image.Sha)
}

func (image *Image) TarFilePath() string {
	return fmt.Sprintf("./tmp/%s.tar", image.ShaName())
}

func (image *Image) URLEncodedName() string {
	return url.QueryEscape(image.ShaName())
}

// CreateURL returns the URL used for hitting the docker daemon's create endpoint
func (image *Image) CreateURL() string {
	// TODO v1.24 refers to the docker version.  figure out how to avoid hard-coding this
	// TODO can probably use the docker api code for this
	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s", image.URLEncodedName())
	//	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s&tag=%s", image.name, image.tag)
}

// GetURL returns the URL used for hitting the docker daemon's get endpoint
func (image *Image) GetURL() string {
	return fmt.Sprintf("http://localhost/v1.24/images/%s/get", image.URLEncodedName())
}
