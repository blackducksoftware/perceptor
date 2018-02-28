#!/bin/sh
# Copyright (C) 2018 Synopsys, Inc.

export PERCEPTOR_POD_NS="perceptortestns"

# TODO Put in a check here if kubectl cli is present

# Create the Namespace
createNs() {
  WAIT_TIME=$((30))
  # Clean up NS JIC it's still here...
  kubectl get ns | grep $PERCEPTOR_POD_NS | xargs kubectl delete ns
  sleep $WAIT_TIME
  kubectl create -f ./perceptorTestNS.yml
  sleep $WAIT_TIME
  kubectl get ns | grep $PERCEPTOR_POD_NS
  ns_state=$(kubectl get ns | grep $PERCEPTOR_POD_NS)
  if [ -z $ns_state ] ; then
    echo "Error: Namespace $PERCEPTOR_POD_NS not found!"
    exit 1;
  else
    echo "Namespace $PERCEPTOR_POD_NS found, w00t! Moving on..."
  fi
}
# Spin up a Kube POD using busybox
createPod() {
echo "Creating POD..."
kubectl run busybox --image=busybox --namespace=$PERCEPTOR_POD_NS
}


# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Check POD has been annotated with Black Duck
tstAnnotate() {
  WAIT_TIME=$((30))
  echo "Checking for Blackduck POD annotations..."
  sleep $WAIT_TIME
  a_state=$(kubectl describe pod $PERCEPTOR_POD_NS | grep "blackduck")
  if [[ -z $a_state ]]; then
    echo "There appears to be no POD Annoations present."
    exit 1;
  else
    echo "Annoations found!"
  fi
}

createNs
createPod
tstAnnotate
