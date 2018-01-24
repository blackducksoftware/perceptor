package hub

type FetcherInterface interface {
	FetchProjectByName(projectName string) (*Project, error)
}
