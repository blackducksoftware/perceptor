set -e

oc create ns bds-perceptor


oc create -f kube-perceiverserviceaccount.yaml --namespace=bds-perceptor
# allows writing of cluster level metadata for imagestreams
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:bds-perceptor:kube-perceiver-sa

oc create -f mock-perceptor.yaml --namespace=bds-perceptor
oc create -f kube-perceiver.yaml --namespace=bds-perceptor

oc create -f routes.yaml --namespace=bds-perceptor
