package common

type Container struct {
	Image Image
	Name  string
}

func NewContainer(image Image, name string) *Container {
	return &Container{Image: image, Name: name}
}
