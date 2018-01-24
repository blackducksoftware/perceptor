package api

import (
	"net/http"

	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
)

type Responder interface {
	// state of the program
	// this is a funky return type because it's so tightly coupled to prometheus
	GetMetrics(w http.ResponseWriter, r *http.Request)
	GetModel() string

	// perceiver
	AddPod(pod common.Pod)
	UpdatePod(pod common.Pod)
	DeletePod(qualifiedName string)
	GetScanResults() ScanResults

	// scanner
	GetNextImage(func(nextImage NextImage))
	PostFinishScan(job FinishedScanClientJob)
}
