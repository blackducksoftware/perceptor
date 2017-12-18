package openshift

import (
	"time"

	"github.com/prometheus/common/log"
)

func main() {

}

func getPodsWatcher(podsSink chan<- OcPodsInfo) {
	for {
		pods := getPods()
		podsSink <- pods
		time.Sleep(2 * time.Second)
		log.Info("get pods watcher")
	}
}

/*
func startScans(model *model.Model, hub *scanner.MockHub, podsSource <-chan OcPodsInfo) { //, newContainerImagesSink chan<- []string) {
	for {
		select {
		case podsInfo := <-podsSource:
			newContainerIds := AddPods(model, &podsInfo)
			hub.scanImages(newContainerIds)
			log.Info("start scans", "[", strings.Join(newContainerIds, ", "), "]")
		}
	}
}
*/
