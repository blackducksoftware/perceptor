package kube

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"encoding/json"

	"github.com/golang/glog" // TODO replace these calls with logrus calls
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

type UpdatePod struct {
	Old v1.Pod
	New v1.Pod
}

type KubeAPIClient struct {
	controller *Controller
	clientset  kubernetes.Clientset
	addPod     chan<- v1.Pod
	updatePod  chan<- UpdatePod
	deletePod  chan<- string
	stop       chan struct{}
}

func (client *KubeAPIClient) startMonitoringPods() {
	go client.controller.Run(1, client.stop)
}

func (client *KubeAPIClient) stopMonitoringPods() {
	close(client.stop)
}

// AddPodAnnotation adds an annotation to a pod in openshift/kube
func (client *KubeAPIClient) AddPodAnnotation(pod *v1.Pod, key string, value string) {
	p, err := client.clientset.CoreV1().Pods(pod.GetNamespace()).Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		panic(err)
	}
	annotations := p.GetAnnotations()
	annotations[key] = value
	p.SetAnnotations(annotations)
	log.Infof("add pod annotation: %v, %s", annotations, pod.GetName())
	updated, err := client.clientset.CoreV1().Pods(pod.GetNamespace()).Update(p)
	if err != nil {
		panic("unable to update pod: " + err.Error())
	} else {
		log.Info("successfully updated pod, %v", updated)
	}
}

func (client *KubeAPIClient) ClearPodAnnotations(pod *v1.Pod) {
	spec := pod.Spec
	container := spec.Containers[0]
	log.Info("container: %v", container)
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	// TODO if `p` is the same type as `pod`, why do we need `p`?
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		panic(err)
	}
	p.SetAnnotations(make(map[string]string))
	_, err = pods.Update(p)
	if err != nil {
		panic("unable to clear pod annotations")
	}
}

func (client *KubeAPIClient) ClearBlackDuckPodAnnotations(pod *v1.Pod) {
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	// TODO if `p` is the same type as `pod`, why do we need `p`?
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		panic(err)
	}
	annotations := p.GetAnnotations()
	delete(annotations, "BlackDuck")
	p.SetAnnotations(annotations)
	_, err = pods.Update(p)
	if err != nil {
		panic("unable to clear BlackDuck pod annotations")
	}
}

// GetBlackDuckPodAnnotations cooperates with SetBlackDuckPodAnnotations,
// which serialize and deserialize a JSON
// dictionary, in the annotations map, under the "BlackDuck" key.
// Rationale:
//   1. to support a rich model
//   2. to avoid stomping on other annotations that have nothing to do
//      with Black Duck
func (client *KubeAPIClient) GetBlackDuckPodAnnotations(pod *v1.Pod) BlackDuckAnnotations {
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		panic(err)
	}
	annotations := p.GetAnnotations()
	// get the JSON string
	var bdString string
	bdString, ok := annotations["BlackDuck"]
	if !ok {
		return *NewBlackDuckAnnotations()
	}
	// string -> BlackDuckAnnotations
	var bdAnnotations BlackDuckAnnotations
	err = json.Unmarshal([]byte(bdString), &bdAnnotations)
	if err != nil {
		message := fmt.Sprintf("unable to Unmarshal BlackDuckAnnotations: %s", err.Error())
		log.Error(message)
		return *NewBlackDuckAnnotations()
	}
	return bdAnnotations
}

func (client *KubeAPIClient) SetPodAnnotations(pod *v1.Pod, bdAnnotations BlackDuckAnnotations) error {
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

/*
func (client *KubeAPIClient) AddBlackDuckPodAnnotations(pod *v1.Pod, key string, value string) {
	pods := client.clientset.CoreV1().Pods(pod.GetNamespace())
	// TODO if `p` is the same type as `pod`, why do we need `p`?
	p, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		panic(err)
	}
	annotations := p.GetAnnotations()
	// get the JSON string
	var bdString string
	bdString, ok := annotations["BlackDuck"]
	if !ok {
		bdString = ""
	}
	// string -> BlackDuckAnnotations
	var bdAnnotations BlackDuckAnnotations
	json.Unmarshal([]byte(bdString), &bdAnnotations)
	// add the KV pair
	bdAnnotations.KeyVals[key] = value
	// BlackDuckAnnotations -> string
	jsonBytes, err := json.Marshal(bdAnnotations)
	if err != nil {
		log.Errorf("unable to marshal BlackDuckAnnotations: %s", err.Error())
		return
	}
	// add it into the annotations map
	annotations["BlackDuck"] = string(jsonBytes)
	// update the pod
	p.SetAnnotations(annotations)
	_, err = pods.Update(p)
	if err != nil {
		log.Errorf("unable to update pod: %s", err.Error())
		panic("unable to update BlackDuck pod annotations")
	}
}
*/

func NewKubeAPIClient(addPod chan<- v1.Pod, updatePod chan<- UpdatePod, deletePod chan<- string) *KubeAPIClient {
	kubeconfig := "/Users/mfenwick/.kube/config"
	master := "https://34.227.56.110.xip.io:8443"
	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}

	// create the pod watcher
	namespace := v1.NamespaceAll // v1.NamespaceDefault
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", namespace, fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

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
				addPod <- *pod
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
				updatePod <- UpdatePod{New: *newPod, Old: *oldPod}
			}
			log.Infof("update -- %s", newKey)
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
				// TODO do we need to copy obj to ensure it isn't changed by something else?
				deletePod <- key
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

	controller := NewController(queue, indexer, informer)

	client := KubeAPIClient{
		controller: controller,
		clientset:  *clientset,
		addPod:     addPod,
		updatePod:  updatePod,
		deletePod:  deletePod,
		stop:       make(chan struct{}),
	}
	client.startMonitoringPods()

	return &client
}
