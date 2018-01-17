package clustermanager

import (
	common "bitbucket.org/bdsengineering/perceptor/pkg/common"

	"k8s.io/api/core/v1"
)

func NewPod(kubePod *v1.Pod) *common.Pod {
	containers := []common.Container{}
	for _, newCont := range kubePod.Spec.Containers {
		addedCont := common.NewContainer(common.Image(newCont.Image), newCont.Name)
		containers = append(containers, *addedCont)
	}
	return common.NewPod(kubePod.Name, string(kubePod.UID), kubePod.Namespace, containers)
}
