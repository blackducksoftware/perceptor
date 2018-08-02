#!/bin/bash

NS=$1

kubectl delete -f federator-configmap.yaml -n $NS
kubectl delete -f federator.yaml -n $NS
kubectl delete -f federator-service.yaml -n $NS
kubectl delete service federator-public -n $NS
