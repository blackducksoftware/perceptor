package scanner

type Path int

const (
	PathGetNextImage    = iota
	PathPostScanResults = iota
)

type HttpResult struct {
	StatusCode int
	Path       Path
}
