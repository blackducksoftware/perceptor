package core

type Container struct {
	Image string
	Name  string
}

func NewContainer(image string, name string) *Container {
	return &Container{
		Image: image,
		Name:  name,
	}
}
