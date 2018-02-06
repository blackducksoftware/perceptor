package core

import (
	"testing"

	"bitbucket.org/bdsengineering/perceptor/pkg/api"
	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"
)

func TestMetrics(t *testing.T) {
	m := newMetrics()
	if m == nil {
		t.Error("expected m to be non-nil")
	}

	m.addImage(common.Image{})
	m.addPod(common.Pod{})
	m.allPods(api.AllPods{})
	m.deletePod("abcd")
	m.getNextImage()
	m.getScanResults()
	// TODO not good for testing
	// m.httpError(request, err)
	// m.httpNotFound(request)
	m.postFinishedScan()
	m.updateModel(Model{Images: map[common.Image]*ImageScanResults{
		common.Image{}: &ImageScanResults{ScanStatus: ScanStatusInQueue, ScanResults: nil},
	}})
	m.updatePod(common.Pod{})

	message := "finished test case"
	t.Log(message)
	log.Info(message)
}
