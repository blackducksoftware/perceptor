package kube

type BlackDuckAnnotations struct {
  Containers map[string]Container
  // TODO remove KeyVals, this is just for testing, to be able
  // to jam random stuff somewhere
  KeyVals map[string]string
}

func NewBlackDuckAnnotations() *BlackDuckAnnotations {
  return &BlackDuckAnnotations {
    Containers: make(map[string]Container),
    KeyVals: make(map[string]string),
  }
}

type Container struct {
  Image string
  // vulnerabilities ?
}
