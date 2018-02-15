set -e

# Create the namespace to install all containers
oc create ns bds-perceptor

# Create the serviceaccount for perceptor-scanner to talk with Docker
oc create -f perceptor-scanner-serviceaccount.yaml --namespace=bds-perceptor

# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-scanner-sa

# Create the perceptor container
oc create -f perceptor.yaml --namespace=bds-perceptor

# Create the perceptor-scanner container
oc create -f perceptor-scanner.yaml --namespace=bds-perceptor

#oc create -f routes.yaml --namespace=bds-perceptor
