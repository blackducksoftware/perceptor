set -e

oc create ns perceptor-scan
oc create -f serviceaccount.yaml --namespace=perceptor-scan

# allows writing of cluster level metadata for imagestreams
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:perceptor-scan:perceptor-scan
# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:perceptor-scan:perceptor-scan

oc create -f perceptor.yaml --namespace=perceptor-scan
