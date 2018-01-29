package core

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"bitbucket.org/bdsengineering/perceptor/pkg/hub"
)

type HubImageScan struct {
	Image common.Image
	Scan  *hub.ImageScan
}
