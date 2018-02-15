set -e

# Create the namespace to install all containers
oc create ns bds-perceptor

oc project bds-perceptor

# Create the serviceaccount for perceptor-scanner to talk with Docker
oc create -f perceptor-scanner-serviceaccount.yaml

# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-scanner-sa

# Create the perceptor container
oc create -f perceptor.yaml

# Create the perceptor-scanner container
oc create -f perceptor-scanner.yaml

# Create the openshift-perceiver service account
oc create serviceaccount openshift-perceiver

# following allows us to write cluster level metadata for imagestreams
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount::openshift-perceiver

# Create the perceptor-scanner container
oc create -f openshift-perceiver.yaml

#oc create -f routes.yaml --namespace=bds-perceptor
