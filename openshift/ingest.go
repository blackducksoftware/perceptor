package openshift

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

/*
 * OcPods: a pod model from `oc get pods -o json --all-namespaces`
 */
type OcPod struct {
	//		APIVersion string
	// Kind     string
	Metadata struct {
		Name      string
		UID       string
		Namespace string
	}
	Spec struct {
		//			DNSPolicy string
		Containers []struct {
			Image string
			Name  string
		}
	}
}

type OcPodsInfo struct {
	Items []OcPod
}

func parse(s string) OcPodsInfo {
	decoder := json.NewDecoder(strings.NewReader(s))
	var pods OcPodsInfo
	if err := decoder.Decode(&pods); err != nil {
		fmt.Println("deserialization error: ", err)
		panic("whoops!  couldn't parse -- ") // + string(err))
	}
	return pods
}

func getPods() OcPodsInfo {
	cmd := exec.Command("oc", "get", "pods", "-o", "json", "--all-namespaces")
	fmt.Println("running command 'oc get pods ...'")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return parse(fmt.Sprintf("%s", stdoutStderr))
}

/* TODO uncomment if necessary
func addPods(model *model.Model, pods *OcPodsInfo) []string {
	newContainerImages := []string{}
	for _, incomingPod := range pods.Items {
		uid := incomingPod.Metadata.UID
		if _, ok := model.Pods[uid]; !ok {
			pod := model.addPod(uid, incomingPod.Metadata.Namespace)
			for _, cont := range incomingPod.Spec.Containers {
				pod.containerImages = append(pod.containerImages, cont.Image)
				if !model.hasContainerOfImage(cont.Image) {
					newContainerImages = append(newContainerImages, cont.Image)
				}
			}
		}
	}
	return newContainerImages
}
*/
