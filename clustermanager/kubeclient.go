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

func NewPod(kubePod *v1.Pod) *Pod {
	pod := Pod{
		Name:        kubePod.Name,
		Namespace:   kubePod.Namespace,
		Annotations: kubePod.Annotations,
		Spec:        *NewSpec(&kubePod.Spec),
		UID:         string(kubePod.UID),
	}
	return &pod
}

func NewSpec(kubeSpec *v1.PodSpec) *Spec {
	containers := []Container{}
	for _, kubeCont := range kubeSpec.Containers {
		containers = append(containers, *NewContainer(&kubeCont))
	}
	return &Spec{
		Containers: containers,
	}
}

func NewContainer(kubeCont *v1.Container) *Container {
	return &Container{
		Image: kubeCont.Image,
		Name:  kubeCont.Name,
	}
}

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

func (client *KubeClient) GetAnnotations(pod *v1.Pod) (map[string]string, error) {
	pods := client.clientset.CoreV1().Pods(pod.Namespace)
	kubePod, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return nil, err
	}
	return kubePod.GetAnnotations(), nil
}

func (client *KubeClient) SetAnnotations(pod *v1.Pod, annotations map[string]string) error {
	pods := client.clientset.CoreV1().Pods(pod.Namespace)
	kubePod, err := pods.Get(pod.Name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return err
	}
	kubePod.SetAnnotations(annotations)
	_, err = pods.Update(kubePod)
	if err != nil {
		log.Errorf("unable to update pod: %s", err.Error())
	}
	return err
}

func (client *KubeClient) AddAnnotation(pod *v1.Pod, key string, value string) error {
	annotations, err := client.GetAnnotations(pod)
	if err != nil {
		return err
	}
	// TODO should a copy of annotations be made first?
	annotations[key] = value
	return client.SetAnnotations(pod, annotations)
}

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
	kubePod, err := pods.Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return err
	}
	annotations := kubePod.GetAnnotations()
	delete(annotations, "BlackDuck")
	kubePod.SetAnnotations(annotations)
	_, err = pods.Update(kubePod)
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
func (client *KubeClient) GetBlackDuckPodAnnotations(namespace string, name string) (*BlackDuckAnnotations, error) {
	pods := client.clientset.CoreV1().Pods(namespace)
	kubePod, err := pods.Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return nil, err
	}
	annotations := kubePod.GetAnnotations()
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
		return NewBlackDuckAnnotations(), nil
		//		return nil, err
	}
	if bdAnnotations.ImageAnnotations == nil {
		bdAnnotations.ImageAnnotations = make(map[string]ImageAnnotation)
	}
	if bdAnnotations.KeyVals == nil {
		bdAnnotations.KeyVals = make(map[string]string)
	}
	return &bdAnnotations, nil
}

func (client *KubeClient) SetBlackDuckPodAnnotations(namespace string, name string, bdAnnotations BlackDuckAnnotations) error {
	pods := client.clientset.CoreV1().Pods(namespace)
	kubePod, err := pods.Get(name, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("unable to get pod: %s", err.Error())
		return err
	}
	annotations := kubePod.GetAnnotations()
	// BlackDuckAnnotations -> string
	jsonBytes, err := json.Marshal(bdAnnotations)
	if err != nil {
		log.Errorf("unable to marshal BlackDuckAnnotations: %s", err.Error())
		return err
	}
	// add it into the annotations map
	annotations["BlackDuck"] = string(jsonBytes)
	// update the pod
	kubePod.SetAnnotations(annotations)
	_, err = pods.Update(kubePod)
	if err != nil {
		log.Errorf("unable to update pod: %s", err.Error())
	}
	return err
}

// Some extra, maybe useless methods

func (client *KubeClient) clearBlackDuckPodAnnotationsWithPod(pod Pod) error {
	return client.ClearBlackDuckPodAnnotations(pod.Namespace, pod.Name)
}

func (client *KubeClient) getBlackDuckPodAnnotationsWithPod(pod Pod) (*BlackDuckAnnotations, error) {
	return client.GetBlackDuckPodAnnotations(pod.Namespace, pod.Name)
}

func (client *KubeClient) setBlackDuckPodAnnotationsWithPod(pod Pod, bdAnnotations BlackDuckAnnotations) error {
	return client.SetBlackDuckPodAnnotations(pod.Namespace, pod.Name, bdAnnotations)
}

// End extra, maybe useless methods

func NewKubeClient(masterURL string, kubeconfigPath string) (*KubeClient, error) {
	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
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
	// TODO set the namespace
	// namespace := v1.NamespaceAll
	namespace := v1.NamespaceDefault
	// namespace := "blackduck-scan"
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
			log.Infof("add -- %s", key)
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
