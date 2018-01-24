set -e

oc create ns bds-perceptor
oc create -f perceptor-scanner-serviceaccount.yaml --namespace=bds-perceptor

# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-scanner-sa

oc create -f perceptor-scanner.yaml --namespace=bds-perceptor
