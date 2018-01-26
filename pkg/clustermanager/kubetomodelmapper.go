package clustermanager

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"
	"github.com/prometheus/common/log"

	"k8s.io/api/core/v1"
)

func NewPod(kubePod *v1.Pod) *common.Pod {
	containers := []common.Container{}
	for _, newCont := range kubePod.Status.ContainerStatuses {
		name, sha, err := ParseImageIDString(newCont.ImageID)
		if err != nil {
			log.Errorf("unable to parse kubernetes imageID string %s: %s", newCont.ImageID, err.Error())
			continue
		}
		addedCont := common.NewContainer(*common.NewImage(name, sha, newCont.Image), newCont.Name)
		containers = append(containers, *addedCont)
	}
	return common.NewPod(kubePod.Name, string(kubePod.UID), kubePod.Namespace, containers)
}
