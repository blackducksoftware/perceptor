package core

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

// TODO may want to get rid of this ... is it really necessary?
type ImageScanComplete struct {
	AffectedPods []Pod
	Image        common.Image
	ScanResults  ScanResults
}
