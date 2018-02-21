#!/bin/sh
# Copyright (C) 2018 Synopsys, Inc.

export PERCEPTOR_POD_NS="perceptortestns"
export REGISTRY_PORT=":5000"

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
    echo "ERROR: Namespace $PERCEPTOR_POD_NS not found!"
    exit 1;
  else
    echo "Namespace $PERCEPTOR_POD_NS found, w00t! Moving on..."
  fi
}
# Spin up a POD using oc run busybox
# TestRail Test Case C7556
createPod() {
  echo "Test: Creating POD using 'oc run'..."
  oc run busybox --image=busybox --namespace=$PERCEPTOR_POD_NS
  oc project perceptortestns
  my_pod=$(oc get pods | grep -i busybox | cut -d ' ' -f 1)
    if [ -z $my_pod ] ; then
      echo "ERROR: No POD found matching $my_pod!"
      exit 7556;
    else
      echo "POD name $my_pod found, w00t! Moving on..."
    echo "$my_pod"
    fi
}

# TestRail Test Case C7440
createDockerHub() {
  echo "Test: Deploying directly via DockerHUB"
  oc new-project tst-deploy-dockerhub
  oc new-app centos/python-35-centos7~https://github.com/openshift/django-ex.git
  my_pod=$(oc get pods | grep -i django | cut -d ' ' -f 1)
    if [ -z $my_pod ] ; then
      echo "ERROR: No POD found matching $my_pod!"
      exit 7440;
    else
      echo "POD name $my_pod found, , w00t! Moving on..."
    fi
}

# TestRail Test Case C7441
createDockerPull() {
  echo "Test: Deploying with Docker Pull then oc new-app"
  docker pull alpine
  oc new-project tst-docker-pull
  oc new-app docker.io/alpine:latest
  my_pod=$(oc get pods | grep -i alpine | cut -d ' ' -f 1)
  if [ -z $my_pod ] ; then
    echo "ERROR: No POD found matching $my_pod!"
    exit 7441;
  else
    echo "POD name $my_pod found, , w00t! Moving on..."
  fi
}

# Test Rail Test Case C7439
createDockerLoad() {
  echo "Test: Deploying via Docker Load..."
  echo "Pulling hello-world..."
  docker pull hello-world
  echo "Saving hello-world as a tarball..."
  docker save hello-world > /tmp/hello-world.tar
  echo "Loading image via docker load..."
  docker load /tmp/hello-world.tar
  # Login to Openshift
  oc login -u=clustadm -p=devops123!
  oc_token=$(oc whoami -t)
  # Swith to the default project (the registry is here)
  oc project default
  # Let's find the Openshift Registry IP
  # Field 5 is the IP, and we can assume the PORT will always be 5000 and export that
  regIpPort=$(oc get svc | grep docker-registry | cut -d ' ' -f 5)$REGISTRY_PORT
  # Now let's login to the Image Registry
  docker login -u clustadm -e test@synopsys.com -p $oc_token $regIpPort
  # Create a project to push to
  oc new-project pushtest
  # Now let's tag the image
  docker tag docker.io/hello-world:pushtest $regIpPort$REGISTRY_PORT/pushtest/pushtest
  # Now push this puppy to the Registry
  docker push $regIpPort$REGISTRY_PORT/pushtest/pushtest
  # Let's see if the pushtest POD is created...
  my_pod=$(oc get pods | grep -i pushtest | cut -d ' ' -f 1)
  if [ -z $my_pod ] ; then
    echo "ERROR: No POD found matching $my_pod!"
    exit 7439;
  else
    echo "POD name $my_pod found, w00t! Moving on..."
  fi
}

# Test rail Test Case C7445
createS2i() {
  echo "Test: Deploy a Source to Image (S2i)"
  oc new-project puma-test-app
  oc new-app https://github.com/openshift/sti-ruby.git \
  --context-dir=2.0/test/puma-test-app
  my_pod=$(oc get pods | grep -i sti-ruby | cut -d ' ' -f 1)
  if [ -z $my_pod ] ; then
    echo "No POD found matching $my_pod"
    exit 7445;
  else
    echo "POD name $my_pod found, w00t! Moving on...!"
  fi
}

# Test Rail Test Case C7448
createTemplate() {
  echo "Test: Deploy an Image via OpenShift Template"
  oc new-project php
  oc new-app -f /usr/share/openshift/examples/quickstart-templates/rails-postgresql.json
  my_pod=$(oc get pods | grep -i rails | cut -d ' ' -f 1)
  if [ -z $my_pod ] ; then
    echo "No POD found matching $my_pod"
    exit 7448;
  else
    echo "POD name $my_pod found, w00t! Moving on...!"
  fi
}

# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Verify POD has been annotated with "BlackDuck"
tstAnnotate() {
  WAIT_TIME=$((30))
  echo "Checking for BlackDuck POD annotations..."
  sleep $WAIT_TIME
  a_state=$(oc describe pod $my_pod | grep -i BlackDuck)
  if [[ $a_state == "" ]]; then
    echo "ERROR: There appears to be no POD Annoations present!"
    exit $?;
  else
    echo "BlackDuck OpsSight Annoations found! TEST PASS"
  fi
}

x=0

createNs
createPod
tstAnnotate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createPod"
  exit $?
fi

createDockerHub
tstAnnotate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createDockerHub"
  exit $?
fi

createDockerPull
tstAnnotate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createDockerPull"
  exit $?
fi

createDockerLoad
tstAnnotate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createDockerLoad"
  exit $?
fi

createS2i
tstAnnotate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createS2i"
  exit $?
fi

createTemplate
tstAnnotate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createTemplate"
  exit $?
fi
