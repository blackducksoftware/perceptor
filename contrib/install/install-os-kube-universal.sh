#!/bin/bash 

NS=bds-perceptor
KUBECTL="kubectl"

function is_openshift {
	oc version	
	return
}

cleanup() {
	is_openshift
	if ! $(exit $?); then
		echo "assuming kube"
		KUBECTL="kubectl"
		$KUBECTL delete ns $NS
                $KUBECTL delete sa perceptor-sanner-sa
	else 
		KUBECTL="oc"
		if oc get ns | grep -q bds-perceptor ; then
			echo "deleting pereptor project!!!"
			sleep 2
			oc delete project $NS
		fi
		oc delete sa perceptor-scanner-sa
	fi
	while oc get ns | grep -q $NS ; do
	    echo "Waiting for deletion...`$KUBECTL get ns | grep perc` "
	    sleep 1
        done
}

install() {
	is_openshift
	if ! $(exit $?); then
	    echo "assuming kube"
	    $KUBECTL create sa perceptor-scanner-sa
	    $KUBECTL create ns $NS
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
			oc create serviceaccount kube-perceiver-generic
			# following allows us to write cluster level metadata for imagestreams
			oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:bds-perceptor:openshift-perceiver

			# Create the serviceaccount for perceptor-scanner to talk with Docker
			oc create sa perceptor-scanner-sa

			# allows launching of privileged containers for Docker machine access
			oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:perceptor-scanner-sa

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
		$KUBECTL create -f prom.cfg.yml  
		$KUBECTL create -f prometheus-deployment.yaml
	popd
}

cleanup
set -x
set -e
install
set +e
echo "optional install components starting now..."
install-contrib

