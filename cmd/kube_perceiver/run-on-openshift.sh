set -e

oc create ns perceptor-scan
oc create -f kube-perceiverserviceaccount.yaml --namespace=perceptor-scan

# allows writing of cluster level metadata for imagestreams
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:perceptor-scan:kube-perceiver-sa

oc create -f kube-perceiver.yaml --namespace=perceptor-scan
