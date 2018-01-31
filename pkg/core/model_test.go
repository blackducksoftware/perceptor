package core

import (
	"encoding/json"
	"testing"

	"bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/common/log"
)

func TestMarshalModel(t *testing.T) {
	model := Model{ConcurrentScanLimit: 1,
		ImageHubCheckQueue: []common.Image{common.Image{}},
		ImageScanQueue:     []common.Image{},
		Images:             map[common.Image]*ImageScanResults{},
		Pods:               map[string]common.Pod{}}
	jsonBytes, err := json.Marshal(model)
	jb, e := model.MarshalJSON()
	log.Infof("JSON: %s\n%v", string(jb), e)
	if err != nil {
		t.Errorf("unable to marshal %v as JSON: %v", model, err)
		t.Fail()
		return
	}
	expectedString := `{"Pods":{},"Images":{},"ImageScanQueue":[],"ImageHubCheckQueue":[{"Name":"","Sha":"","DockerImage":""}],"ConcurrentScanLimit":1}`
	if string(jsonBytes) != expectedString {
		t.Errorf("expected %s, got %s", expectedString, string(jsonBytes))
	}
}
