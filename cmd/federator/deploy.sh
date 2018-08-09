#!/bin/bash

set -e

NS=$1

kubectl create ns $NS

kubectl create -f federator-configmap.yaml -n $NS
kubectl create -f federator.yaml -n $NS
kubectl create -f federator-service.yaml -n $NS
#kubectl expose service hub-federator --name=federator-public --port=80 --target-port=80 --type=LoadBalancer -n $NS
# for minikube:
kubectl expose service hub-federator --port=3016 --type=NodePort --name=federator-public -n $NS
minikube service federator-public -n $NS --url
