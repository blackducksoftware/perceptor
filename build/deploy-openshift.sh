set -e

oc create ns perceptor-scan


oc create -f serviceaccounts.yaml --namespace=perceptor-scan
# allows writing of cluster level metadata for imagestreams
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:perceptor-scan:kube-perceiver-sa
# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:perceptor-scan:perceptor-sa


oc create -f pods.yaml --namespace=perceptor-scan

oc create -f routes.yaml --namespace=perceptor-scan
