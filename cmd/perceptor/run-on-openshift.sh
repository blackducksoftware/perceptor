set -e

oc create ns bds-perceptor
oc create -f serviceaccount.yaml --namespace=bds-perceptor

# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-sa

oc create -f perceptor.yaml --namespace=bds-perceptor
