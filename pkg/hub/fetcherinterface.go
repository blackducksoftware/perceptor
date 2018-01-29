package hub

import "bitbucket.org/bdsengineering/perceptor/pkg/common"

type FetcherInterface interface {
	FetchScanFromImage(image common.Image) (*ImageScan, error)
}
