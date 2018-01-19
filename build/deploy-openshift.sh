set -e

oc create ns bds-perceptor


oc create -f ../cmd/kube-perceiver/kube-perceiverserviceaccount.yaml --namespace=bds-perceptor
# allows writing of cluster level metadata for imagestreams
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:bds-perceptor:kube-perceiver-sa

oc create -f ../cmd/perceptor/perceptor-serviceaccount.yaml --namespace=bds-perceptor
# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-sa


oc create -f ../cmd/kube-perceiver/kube-perceiver.yaml --namespace=bds-perceptor
oc create -f ../cmd/perceptor/perceptor.yaml --namespace=bds-perceptor


oc create -f routes.yaml --namespace=bds-perceptor
