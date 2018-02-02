package core

import (
  "bitbucket.org/bdsengineering/perceptor/pkg/common"
  "bitbucket.org/bdsengineering/perceptor/pkg/api"
  log "github.com/sirupsen/logrus"
)

type action interface {
  apply(model Model) Model
}


type addPod struct {
  pod common.Pod
}

func (a addPod) apply(model Model) Model {
  model.AddPod(a.pod)
	return model
}


type updatePod struct {
  pod common.Pod
}

func (u updatePod) apply(model Model) Model {
  model.AddPod(u.pod)
	return model
}


type deletePod struct {
  podName string
}

func (d deletePod) apply(model Model) Model {
  _, ok := model.Pods[d.podName]
	if !ok {
		log.Warnf("unable to delete pod %s, pod not found", d.podName)
		return model
	}
	delete(model.Pods, d.podName)
	return model
}


type addImage struct {
  image common.Image
}

func (a addImage) apply(model Model) Model {
  model.AddImage(a.image)
	return model
}


type allPods struct {
  pods []common.Pod
}

func (a allPods) apply(model Model) Model {
  model.Pods = map[string]common.Pod{}
	for _, pod := range a.pods {
		model.AddPod(pod)
	}
	return model
}


type getNextImage struct {
  continuation func(image *common.Image)
}

func (g getNextImage) apply(model Model) Model {
  log.Infof("looking for next image to scan with concurrency limit of %d, and %d currently in progress", model.ConcurrentScanLimit, model.inProgressScanCount())
	image := model.getNextImageFromScanQueue()
	g.continuation(image)
	return model
}


type finishScanClient struct {
  job api.FinishedScanClientJob
}

func (f finishScanClient) apply(model Model) Model {
  newModel := model
	log.Infof("finished scan client job action: error was empty? %t, %+v", f.job.Err == "", f.job.Image)
	if f.job.Err == "" {
		newModel.finishRunningScanClient(f.job.Image)
	} else {
		newModel.errorRunningScanClient(f.job.Image)
	}
	return newModel
}
