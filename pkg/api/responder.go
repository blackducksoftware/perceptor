package httpserver

import (
	"net/http"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

type Responder interface {
	// state of the program
	//   these have funky return types because:
	//    - GetMetrics is so tightly coupled to prometheus
	//    - I don't feel like exposing the type of the model to this package (at least, for now)
	GetMetrics(w http.ResponseWriter, r *http.Request)
	GetModel(w http.ResponseWriter, r *http.Request)

	// perceiver
	AddPod(pod common.Pod)
	UpdatePod(pod common.Pod)
	DeletePod(qualifiedName string)
	// Image(w http.ResponseWriter, r *http.Request, image common.Image)
	GetScanResults() ScanResults

	// scanner
	// TODO GetNextImage() string
	// TODO PostFinishScan(image string, success bool) // or instead of bool, string? enum? need to know what the result was
}
