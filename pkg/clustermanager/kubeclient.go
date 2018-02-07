/*
Copyright (C) 2018 Black Duck Software, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package clustermanager

import (
	"fmt"

	"github.com/blackducksoftware/perceptor/pkg/common"
	log "github.com/sirupsen/logrus"

	"encoding/json"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// KubeClient is an implementation of the Client interface for kubernetes
type KubeClient struct {
	controller *KubeController
	clientset  kubernetes.Clientset
	podAdd     chan AddPod
	podUpdate  chan UpdatePod
	podDelete  chan DeletePod
	stop       chan struct{}
}

func (client *KubeClient) startMonitoringPods() {
	go client.controller.Run(1, client.stop)
}

func (client *KubeClient) stopMonitoringPods() {
	close(client.stop)
}

func (client *KubeClient) PodAdd() <-chan AddPod {
	return client.podAdd
}

func (client *KubeClient) PodUpdate() <-chan UpdatePod {
	return client.podUpdate
}

func (client *KubeClient) PodDelete() <-chan DeletePod {
	return client.podDelete
}

// BEGIN POC methods

func (client *KubeClient) GetPod(namespace string, name string) (*v1.Pod, error) {
	return client.clientset.CoreV1().Pods(namespace).Get(name, meta_v1.GetOptions{})
}

func (client *KubeClient) UpdatePod(pod *v1.Pod) error {
	pods := client.clientset.CoreV1().Pods(pod.Namespace)
	_, err := pods.Update(pod)
	return err
}

// GetPodsForNamespace returns an empty slice if the namespace doesn't exist.
func (client *KubeClient) GetPodsForNamespace(namespace string) ([]v1.Pod, error) {
	options := meta_v1.ListOptions{}
	podList, err := client.clientset.CoreV1().Pods(namespace).List(options)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (client *KubeClient) GetPods() ([]v1.Pod, error) {
	options := meta_v1.ListOptions{}
	namespaceList, err := client.clientset.CoreV1().Namespaces().List(options)
	if err != nil {
		return nil, err
	}
	pods := []v1.Pod{}
	for _, namespace := range namespaceList.Items {
		// log.Infof("checking name %s (namespace %s)\n", namespace.Name, namespace.Namespace)
		podSlice, err := client.GetPodsForNamespace(namespace.Name)
		if err != nil {
			return nil, err
		}
		// log.Infof("found %d pods for namespace %s", len(podSlice), namespace.Name)
		for _, pod := range podSlice {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

// END POC methods

func (client *KubeClient) ClearBlackDuckPodAnnotations(namespace string, name string) error {
	pods := client.clientset.CoreV1().Pods(namespace)
	podName := fmt.Sprintf("%s:%s", namespace, name)
	kubePod, err := pods.Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod %s: %s", podName, err.Error())
		return err
	}
	annotations := kubePod.GetAnnotations()
	delete(annotations, "BlackDuck")
	kubePod.SetAnnotations(annotations)
	_, err = pods.Update(kubePod)
	if err != nil {
		log.Errorf("unable to clear BlackDuck annotations for pod %s: %s", podName, err.Error())
		return err
	}
	return nil
}

// GetBlackDuckPodAnnotations cooperates with SetBlackDuckPodAnnotations,
// which serialize and deserialize a JSON
// dictionary, in the annotations map, under the "BlackDuck" key.
// Rationale:
//   1. to support a rich model
//   2. to avoid stomping on other annotations that have nothing to do
//      with Black Duck
func (client *KubeClient) GetBlackDuckPodAnnotations(namespace string, name string) (*BlackDuckAnnotations, error) {
	pods := client.clientset.CoreV1().Pods(namespace)
	podName := fmt.Sprintf("%s:%s", namespace, name)
	kubePod, err := pods.Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod %s: %s", podName, err.Error())
		return nil, err
	}
	annotations := kubePod.GetAnnotations()
	// get the JSON string
	var bdString string
	bdString, ok := annotations["BlackDuck"]
	if !ok {
		return NewBlackDuckAnnotations(0, 0, ""), nil
	}
	// string -> BlackDuckAnnotations
	var bdAnnotations BlackDuckAnnotations
	err = json.Unmarshal([]byte(bdString), &bdAnnotations)
	if err != nil {
		message := fmt.Sprintf("unable to Unmarshal BlackDuckAnnotations for pod %s: %s", podName, err.Error())
		log.Error(message)
		return NewBlackDuckAnnotations(0, 0, ""), nil
		//		return nil, err
	}
	if bdAnnotations.KeyVals == nil {
		bdAnnotations.KeyVals = make(map[string]string)
	}
	return &bdAnnotations, nil
}

func (client *KubeClient) SetBlackDuckPodAnnotations(namespace string, name string, bdAnnotations BlackDuckAnnotations) error {
	pods := client.clientset.CoreV1().Pods(namespace)
	podName := fmt.Sprintf("%s:%s", namespace, name)
	kubePod, err := pods.Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod %s: %s", podName, err.Error())
		return err
	}
	annotations := kubePod.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	// BlackDuckAnnotations -> string
	jsonBytes, err := json.Marshal(bdAnnotations)
	if err != nil {
		log.Errorf("unable to marshal BlackDuckAnnotations for pod %s: %s", podName, err.Error())
		return err
	}
	// add it into the annotations map
	annotations["BlackDuck"] = string(jsonBytes)
	// update the pod
	kubePod.SetAnnotations(annotations)
	_, err = pods.Update(kubePod)
	if err != nil {
		log.Errorf("unable to update annotations for pod %s: %s", podName, err.Error())
	}
	return err
}

// Some extra, maybe useless methods

// func (client *KubeClient) clearBlackDuckPodAnnotationsWithPod(pod common.Pod) error {
// 	return client.ClearBlackDuckPodAnnotations(pod.Namespace, pod.Name)
// }
//
// func (client *KubeClient) getBlackDuckPodAnnotationsWithPod(pod common.Pod) (*BlackDuckAnnotations, error) {
// 	return client.GetBlackDuckPodAnnotations(pod.Namespace, pod.Name)
// }
//
// func (client *KubeClient) setBlackDuckPodAnnotationsWithPod(pod common.Pod, bdAnnotations BlackDuckAnnotations) error {
// 	return client.SetBlackDuckPodAnnotations(pod.Namespace, pod.Name, bdAnnotations)
// }

// End extra, maybe useless methods

// GetAllPods asks for the kubernetes APIServer for all of its pods
// across all namespaces.
func (client *KubeClient) GetAllPods() ([]common.Pod, error) {
	pods := []common.Pod{}
	kubePods, err := client.clientset.CoreV1().Pods(v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, kubePod := range kubePods.Items {
		pods = append(pods, *NewPod(&kubePod))
	}
	return pods, nil
}

// NewKubeClientFromCluster instantiates a KubeClient using configuration
// pulled from the cluster.
func NewKubeClientFromCluster() (*KubeClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("unable to build config from cluster: %s", err.Error())
		return nil, err
	}
	return newKubeClientHelper(config)
}

// NewKubeClient instantiates a KubeClient using a master URL and
// a path to a kubeconfig file.
func NewKubeClient(masterURL string, kubeconfigPath string) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		log.Errorf("unable to build config from flags: %s", err.Error())
		return nil, err
	}

	return newKubeClientHelper(config)
}

func newKubeClientHelper(config *rest.Config) (*KubeClient, error) {
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("unable to create kubernetes clientset: %s", err.Error())
		return nil, err
	}

	namespace := v1.NamespaceAll

	// create the pod watcher
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", namespace, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	podAdd := make(chan AddPod)
	podUpdate := make(chan UpdatePod)
	podDelete := make(chan DeletePod)

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Pod than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				// TODO do we need to copy obj to ensure it isn't changed by something else?
				pod := obj.(*v1.Pod)
				podAdd <- AddPod{New: *NewPod(pod)}
			} else {
				log.Errorf("error getting key: %s", err.Error())
			}
			log.Infof("kubeclient add pod -- %s", key)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			newKey, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(newKey)
				// TODO do we need to copy old and new to ensure they aren't changed by something else?
				newPod := new.(*v1.Pod)
				oldPod := old.(*v1.Pod)
				podUpdate <- UpdatePod{New: *NewPod(newPod), Old: *NewPod(oldPod)}
			} else {
				log.Errorf("error getting key: %s", err.Error())
			}
			log.Infof("kubeclient update pod -- %s", newKey)
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				podDelete <- DeletePod{QualifiedName: key}
			} else {
				log.Errorf("error getting key: %s", err.Error())
			}
			log.Infof("kubeclient delete pod -- %s", key)
		},
	}, cache.Indexers{})

	controller := NewKubeController(queue, indexer, informer)

	client := KubeClient{
		controller: controller,
		clientset:  *clientset,
		podAdd:     podAdd,
		podUpdate:  podUpdate,
		podDelete:  podDelete,
		stop:       make(chan struct{}),
	}
	client.startMonitoringPods()

	return &client, nil
}
