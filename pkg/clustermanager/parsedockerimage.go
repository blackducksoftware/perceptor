package clustermanager

import (
	"fmt"
	"regexp"

	dockerreference "github.com/docker/distribution/reference"
)

var prefix = regexp.MustCompile("docker-pullable://")
var tagRegexp = regexp.MustCompile(":(" + dockerreference.TagRegexp.String() + ")$")

// not sure why this doesn't work:
// var digestRegexp = regexp.MustCompile("@sha256:(" + dockerreference.DigestRegexp.String() + ")$")
var digestRegexp = regexp.MustCompile("@sha256:([a-zA-Z0-9]+)$")

// ParseImageString parses an image pulled from kubernetes.
// Example image:
//   registry.kipp.blackducksoftware.com/blackducksoftware/hub-registration:4.3.0
// TODO: delete this code, it doesn't work when the tag is missing, and it'd
//   probably be really hard to get it working in this terrible fashion
func ParseImageString(image string) (string, string, error) {
	match := tagRegexp.FindStringSubmatchIndex(image)
	if len(match) != 4 {
		return "", "", fmt.Errorf("unable to match tag regex <%s> to input <%s>", tagRegexp.String(), image)
	}
	name := image[:match[0]]
	tag := image[match[2]:match[3]]
	return name, tag, nil
}

// ParseImageIDString parses an ImageID pulled from kubernetes.
// Example image id:
//   docker-pullable://registry.kipp.blackducksoftware.com/blackducksoftware/hub-registration@sha256:cb4983d8399a59bb5ee6e68b6177d878966a8fe41abe18a45c3b1d8809f1d043
func ParseImageIDString(imageID string) (string, string, error) {
	str := imageID
	match := prefix.FindStringIndex(str)
	if len(match) > 0 && match[0] == 0 {
		str = str[match[1]:]
	} else {
		return "", "", fmt.Errorf("could not find prefix <%s> in <%s>", prefix, imageID)
	}
	match2 := digestRegexp.FindStringSubmatchIndex(str)
	if len(match2) != 4 {
		return "", "", fmt.Errorf("unable to match digestRegexp regex <%s> to input <%s>", digestRegexp.String(), str)
	}
	name := str[:match2[0]]
	digest := str[match2[2]:match2[3]]
	return name, digest, nil
}
