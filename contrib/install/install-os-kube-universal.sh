#!/bin/bash

set +x
NS=bds-perceptor
KUBECTL="kubectl"

function is_openshift {
	if `which oc` ; then
		oc version
		return 0
	else
		return 1
	fi
	return 1
}

cleanup() {
	is_openshift
	if ! $(exit $?); then
		echo "assuming kube"
		KUBECTL="kubectl"
		kubectl delete ns $NS
    kubectl delete sa perceptor-scanner-sa -n $NS
	else
		KUBECTL="oc"
		if oc get ns | grep -q bds-perceptor ; then
			echo "deleting pereptor project!!!"
			sleep 2
			oc delete project $NS
		fi
		oc delete sa perceptor-scanner-sa
	fi
	while $KUBECTL get ns | grep -q $NS ; do
	  echo "Waiting for deletion...`$KUBECTL get ns | grep $NS` "
	  sleep 1
  done
}

install() {
	SCC="add-scc-to-user"
	ROLE="add-role-to-user"
	CLUSTER="add-cluster-role-to-user"
  SYSTEM_SA="system:serviceaccount"

  PERCEPTOR_SC="perceptor-scanner"
	NS_SA="${SYSTEM_SA}:${NS}"
	SCANNER_SA="${NS_SA}:${PERCEPTOR_SCANNER}"

  OS_PERCEIVER="openshift-perceiver"
	OS_PERCEIVER_SA="${NS_SA}:${OS_PERCEIVER}"

  KUBE_PERCEIVER="kube-generic-perceiver"
	KUBE_PERCEIVER_SA="${NS_SA}:${KUBE_PERCEIVER}"

	is_openshift
	if ! $(exit $?); then
    echo "assuming kube"
	  kubectl create ns $NS
	  kubectl create sa perceptor-scanner-sa -n $NS
	  kubectl create sa kube-generic-perceiver -n $NS
  else
	  set -e
	  KUBECTL="oc"
	  echo "Detected openshift... setting up "
	  # Create the namespace to install all containers
	  oc new-project $NS

	  pushd openshift/
	  # Create the openshift-perceiver service account
		oc create serviceaccount openshift-perceiver
		# Create the openshift-perceiver service account
		oc create serviceaccount kube-generic-perceiver
		# following allows us to write cluster level metadata for imagestreams
		oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:bds-perceptor:openshift-perceiver

		# Create the serviceaccount for perceptor-scanner to talk with Docker
		oc create sa perceptor-scanner-sa

		# allows launching of privileged containers for Docker machine access
		oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-scanner-sa

		# following allows us to write cluster level metadata for imagestreams
		oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:bds-perceptor:perceptor-scanner-sa

		# To pull or view all images
		oc policy add-role-to-user view system:serviceaccount::perceptor-scanner-sa

		# Create the perciever for images
		oc create -f openshift-perceiver.yaml
	  popd
	fi

	####
	#### The perceptor core functionality
	####
	pushd kube/
		$KUBECTL create -f kube-perceiver.yaml --namespace=$NS
		$KUBECTL create -f perceptor-scanner.yaml --namespace=$NS
		$KUBECTL create -f perceptor.yaml --namespace=$NS
	popd
}

install-contrib() {
	set -e

	# Deploy a small, local prometheus.  It is only used for scraping perceptor.  Doesnt need fancy ACLs for
	# cluster discovery etc.
	pushd prometheus/
		$KUBECTL create -f prometheus-deployment.yaml --namespace=$NS
	popd
}

cleanup
install
echo "optional install components starting now..."
install-contrib

# IMPORTANT: All the config for perceptor lives here !c
$KUBECTL create -f perceptor.cfg.yml --namespace=$NS
