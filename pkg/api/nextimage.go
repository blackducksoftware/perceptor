package api

import "bitbucket.org/bdsengineering/perceptor/pkg/common"

type NextImage struct {
	Image *common.Image
}

func NewNextImage(image *common.Image) *NextImage {
	return &NextImage{Image: image}
}
