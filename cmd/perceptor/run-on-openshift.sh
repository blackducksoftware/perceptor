set -e

oc create ns bds-perceptor

oc create -f perceptor.yaml --namespace=bds-perceptor
