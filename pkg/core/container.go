package core

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

type Container struct {
	Image common.Image
	Name  string
}

func NewContainer(image common.Image, name string) *Container {
	return &Container{
		Image: image,
		Name:  name,
	}
}
