package clustermanager

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"encoding/json"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
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

func (client *KubeClient) ClearBlackDuckPodAnnotations(pod *v1.Pod) error {
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	// TODO if `p` is the same type as `pod`, why do we need `p`?
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return err
	}
	annotations := p.GetAnnotations()
	delete(annotations, "BlackDuck")
	p.SetAnnotations(annotations)
	_, err = pods.Update(p)
	if err != nil {
		log.Errorf("unable to clear BlackDuck pod annotations: %s", err.Error())
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
func (client *KubeClient) GetBlackDuckPodAnnotations(pod *v1.Pod) (*BlackDuckAnnotations, error) {
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return nil, err
	}
	annotations := p.GetAnnotations()
	// get the JSON string
	var bdString string
	bdString, ok := annotations["BlackDuck"]
	if !ok {
		return NewBlackDuckAnnotations(), nil
	}
	// string -> BlackDuckAnnotations
	var bdAnnotations BlackDuckAnnotations
	err = json.Unmarshal([]byte(bdString), &bdAnnotations)
	if err != nil {
		message := fmt.Sprintf("unable to Unmarshal BlackDuckAnnotations: %s", err.Error())
		log.Error(message)
		return nil, err
	}
	return &bdAnnotations, nil
}

func (client *KubeClient) SetBlackDuckPodAnnotations(pod *v1.Pod, bdAnnotations BlackDuckAnnotations) error {
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	// TODO if `p` is the same type as `pod`, why do we need `p`?
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return err
	}
	annotations := p.GetAnnotations()
	// BlackDuckAnnotations -> string
	jsonBytes, err := json.Marshal(bdAnnotations)
	if err != nil {
		log.Errorf("unable to marshal BlackDuckAnnotations: %s", err.Error())
		return err
	}
	// add it into the annotations map
	annotations["BlackDuck"] = string(jsonBytes)
	// update the pod
	p.SetAnnotations(annotations)
	_, err = pods.Update(p)
	if err != nil {
		log.Errorf("unable to update pod: %s", err.Error())
	}
	return err
}

func NewKubeClient() (*KubeClient, error) {
	kubeconfig := "/Users/mfenwick/.kube/config"
	master := "https://34.227.56.110.xip.io:8443"
	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		log.Errorf("unable to build config from flags: %s", err.Error())
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("unable to create kubernetes clientset: %s", err.Error())
		return nil, err
	}

	// create the pod watcher
	namespace := v1.NamespaceAll // v1.NamespaceDefault
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
				podAdd <- AddPod{New: *pod}
			} else {
				log.Errorf("error getting key: %s", err.Error())
			}
			log.Infof("add -- %s", key)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			newKey, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(newKey)
				// TODO do we need to copy old and new to ensure they aren't changed by something else?
				newPod := new.(*v1.Pod)
				oldPod := old.(*v1.Pod)
				podUpdate <- UpdatePod{New: *newPod, Old: *oldPod}
			} else {
				log.Errorf("error getting key: %s", err.Error())
			}
			log.Infof("update -- %s", newKey)
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				podDelete <- DeletePod{ID: key}
			} else {
				log.Errorf("error getting key: %s", err.Error())
			}
			log.Infof("delete -- %s", key)
		},
	}, cache.Indexers{})

	// TODO delete this example
	/*
		indexer.Add(&v1.Pod{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "mypod",
				Namespace: v1.NamespaceDefault,
			},
		})
	*/

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
