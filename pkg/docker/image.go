package docker

import "fmt"

type Image struct {
	name string
	tag  string
}

func NewImage(name string, tag string) *Image {
	return &Image{name: name, tag: tag}
}

func (image *Image) directory() string {
	// TODO use directory.join or whatever
	return fmt.Sprintf("%s/%s", image.name, image.tag)
}

func (image *Image) path() string {
	return fmt.Sprintf("./tmp/%s/%s", image.name, image.tag)
}

func (image *Image) tarFilePath() string {
	return fmt.Sprintf("%s.tar", image.path())
}

func (image *Image) createURL() string {
	// TODO v1.24 refers to the docker version.  figure out how to avoid hard-coding this
	// TODO can probably use the docker api code for this
	return fmt.Sprintf("http://localhost/v1.24/images/create?fromImage=%s&tag=%s", image.name, image.tag)
}

func (image *Image) getURL() string {
	// TODO we'll leave off user for now, but maybe it should be added back in later ???
	//   the digest could also be added in
	// imageName := fmt.Sprintf("%s%s%s%s%s", image.user, "%2F", image.name, "%3A", image.tag)
	imageName := fmt.Sprintf("%s%s%s", image.name, "%3A", image.tag)
	return fmt.Sprintf("/images/%s/get", imageName)
}
