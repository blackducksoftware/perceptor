set -e

oc create ns perceptor-scan
oc create -f serviceaccount.yaml --namespace=perceptor-scan

# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:perceptor-scan:perceptor-sa

oc create -f perceptor.yaml --namespace=perceptor-scan
