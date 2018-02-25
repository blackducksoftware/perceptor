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
	    kubectl create sa $PERCEPTOR_SCANNER -n $NS
	    kubectl create sa $KUBE_PERCEIVER -n $NS
        else
	    set -e
	    KUBECTL="oc"
	    echo "Detected openshift... setting up "
	    # Create the namespace to install all containers
	    oc new-project $NS

	    pushd openshift/
				# Create the openshift-perceiver service account
				oc create serviceaccount $OS_PERCEIVER
					oc adm policy $CLUSTER cluster-admin $OS_PERCEIVER_SA

				oc create serviceaccount $KUBE_PERCEIVER
					oc adm policy $CLUSTER cluster-admin $K_PERCEIVER_SA

				oc create sa perceptor-scanner-sa
					oc adm policy $SCC privileged $SCANNER_SA
					oc adm policy $CLUSTER cluster-admin $SCANNER_SA

				### To pull or view all images
				### NOTE THAT WE HAVE ** NO ** Namespace defined here !
				oc policy $ROLE view ${SYSTEM_SA}::$PERCEPTOR_SC
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
		$KUBECTL create -f prom.cfg.yml --namespace=$NS
		$KUBECTL create -f prometheus-deployment.yaml --namespace=$NS
	popd
}

# IMPORTANT: All the config for perceptor lives here !c
$KUBECTL create -f perceptor.cfg.yml --namespace=$NS

cleanup
install
echo "optional install components starting now..."
install-contrib
