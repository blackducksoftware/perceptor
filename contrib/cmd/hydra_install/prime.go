/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"k8s.io/api/core/v1"

	resource "k8s.io/apimachinery/pkg/api/resource"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PerceptorRC struct {
	configMap          string
	configMapMountPath string
	name               string
	image              string
	port               int32
	cmd                []string
}

// prime is an experimental installer which bootstraps perceptor and the other
// autobots.
func main() {

	namespace := "bds-perceptor"

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	defaultMem, err := resource.ParseQuantity("2Gi")
	defaultCPU, err := resource.ParseQuantity("500m")

	// this function creates an RC and services that forward to it.
	newRcSvc := func(descriptions []PerceptorRC) (*v1.ReplicationController, []*v1.Service) {

		TheVolumes := []v1.Volume{}
		TheContainers := []v1.Container{}

		for _, desc := range descriptions {
			TheVolumes = append(TheVolumes,
				v1.Volume{
					Name: desc.configMap,
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: desc.configMap,
							},
						},
					},
				})
			TheContainers = append(TheContainers, v1.Container{
				Name:            desc.name,
				Image:           desc.image,
				ImagePullPolicy: "Always",
				Command:         desc.cmd,
				Ports: []v1.ContainerPort{
					v1.ContainerPort{
						ContainerPort: desc.port,
						Protocol:      "TCP",
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    defaultCPU,
						v1.ResourceMemory: defaultMem,
					},
				},
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{
						Name:      desc.configMap,
						MountPath: desc.configMapMountPath,
					},
				},
			})
		}
		rc := &v1.ReplicationController{
			ObjectMeta: v1meta.ObjectMeta{
				Name: descriptions[0].name,
			},
			Spec: v1.ReplicationControllerSpec{
				Selector: map[string]string{"name": descriptions[0].name},
				Template: &v1.PodTemplateSpec{
					ObjectMeta: v1meta.ObjectMeta{
						Labels: map[string]string{"name": descriptions[0].name},
					},
					Spec: v1.PodSpec{
						Volumes:    TheVolumes,
						Containers: TheContainers,
					},
				},
			},
		}

		services := []*v1.Service{}
		for _, desc := range descriptions {
			services = append(services, &v1.Service{
				ObjectMeta: v1meta.ObjectMeta{
					Name: desc.name,
				},
				Spec: v1.ServiceSpec{
					Ports: []v1.ServicePort{
						v1.ServicePort{
							Name: desc.name,
							Port: desc.port,
						},
					},
					Selector: map[string]string{
						"name": desc.name,
					},
				},
			})
		}

		return rc, services
	}

	// perceptor = only one container, very simple.
	rcPCP, svcPCP := newRcSvc([]PerceptorRC{
		PerceptorRC{
			configMap:          "perceptor-config",
			configMapMountPath: "/etc/perceptor",
			name:               "perceptor",
			image:              "gcr.io/gke-verification/blackducksoftware/perceptor:latest",
			port:               3001,
			cmd:                []string{"./perceptor"},
		},
	})

	// perceptorScan = only one container, but will be split into two later?
	rcPCPScan, svcPCPScan := newRcSvc([]PerceptorRC{
		PerceptorRC{
			configMap:          "perceptor-scanner-config",
			configMapMountPath: "/etc/perceptor_scanner",
			name:               "perceptor-scanner",
			image:              "gcr.io/gke-verification/blackducksoftware/perceptor-scanner:latest",
			port:               3003,
			cmd:                []string{"./dependencies/perceptor-scanner"},
		},
	})

	// perceivers
	rcPCVR, svcPCVR := newRcSvc([]PerceptorRC{
		PerceptorRC{
			configMap:          "kube-generic-perceiver-config",
			configMapMountPath: "/etc/perceiver",
			name:               "pod-perceiver",
			image:              "gcr.io/gke-verification/blackducksoftware/pod-perceiver:latest",
			port:               4000,
			cmd:                []string{},
		},
	})

	rcPCVRo, svcPCVRo := newRcSvc([]PerceptorRC{
		PerceptorRC{
			configMap:          "openshift-perceiver-config",
			configMapMountPath: "/etc/perceiver",
			name:               "image-perceiver",
			image:              "gcr.io/gke-verification/blackducksoftware/image-perceiver:latest",
			port:               4000,
			cmd:                []string{},
		},
	})

	_, err = clientset.Core().ReplicationControllers(namespace).Create(rcPCP)
	if err!= nil {
		panic(err)
	}

	for _, svc := range svcPCP {
		_, err = clientset.Core().Services(namespace).Create(svc)
		if err != nil {
			panic(err)
		}
	}

	_, err = clientset.Core().ReplicationControllers(namespace).Create(rcPCPScan)
	if err!= nil {
		panic(err)
	}
	for _, svc := range svcPCPScan {
		_, err = clientset.Core().Services(namespace).Create(svc)
		if err != nil {
			panic(err)
		}
	}

	_, err = clientset.Core().ReplicationControllers(namespace).Create(rcPCVR)
	if err!= nil {
		panic(err)
	}
	for _, svc := range svcPCVR {
		_, err = clientset.Core().Services(namespace).Create(svc)
		if err != nil {
			panic(err)
		}
	}
	_, err = clientset.Core().ReplicationControllers(namespace).Create(rcPCVRo)
	if err!= nil {
		panic(err)
	}
	for _, svc := range svcPCVRo {
		_, err = clientset.Core().Services(namespace).Create(svc)
		if err != nil {
			panic(err)
		}
	}
}
