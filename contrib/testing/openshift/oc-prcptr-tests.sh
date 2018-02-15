# Copyright (C) 2018 Synopsys, Inc.
#!/bin/sh

export PERCEPTOR_POD_NS="perceptortestns"

# TODO Put in a check here if oc cli is present

# Create the Namespace
createNs() {
  WAIT_TIME=$((30))
  # Clean up NS JIC it's still here...
  oc get ns | grep $PERCEPTOR_POD_NS | cut -d ' ' -f 1 | xargs oc delete ns
  sleep $WAIT_TIME
  oc create -f ./perceptorTestNS.yml
  sleep $WAIT_TIME
  oc get ns | grep $PERCEPTOR_POD_NS
  ns_state=$(oc get ns | grep $PERCEPTOR_POD_NS)
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
oc run busybox --image=busybox --namespace=$PERCEPTOR_POD_NS
oc project perceptortestns
my_pod=$(oc get pods | grep -i busybox | cut -d ' ' -f 1)
echo "$my_pod"
}

# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Check POD has been annotated with Black Duck
tstAnnotate() {
  WAIT_TIME=$((30))
  echo "Checking for BlackDuck POD annotations..."
  sleep $WAIT_TIME
  a_state=$(oc describe pod $my_pod | grep -i BlackDuck)
  if [[ $a_state == "" ]]; then
    echo "There appears to be no POD Annoations present."
    exit 1;
  else
    echo "Annoations found!"
  fi
}

createNs
createPod
tstAnnotate
