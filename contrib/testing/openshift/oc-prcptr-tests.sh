#!/bin/bash
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
# Deploy an image using OC RUN
createPod() {
  echo "Test: Creating POD using 'oc run'..."
  oc run busybox --image=busybox --namespace=$PERCEPTOR_POD_NS
  oc project perceptortestns
  my_pod=$(oc get pods | grep -i busybox | cut -d ' ' -f 1)
    if [ -z $my_pod ] ; then
      echo "ERROR: No POD found matching $my_pod!"
      return 1;
    else
      echo "POD name $my_pod found, w00t! Moving on..."
      echo "$my_pod"
    fi
}

# TestRail Test Case C7440
# Creates a new app using Source to Image (S2i), a Builder and Applicatiob
# POD are created in this deployment.
createDockerHub() {
  NS=tst-deploy-dockerhub
  NEW_APP=$1
  PODS=$2
  my_pod=$1
  echo "Test: Deploying directly via DockerHUB with $NEW_APP and we want to see $PODS"
  oc new-project tst-deploy-dockerhub
  oc new-app $NEW_APP
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "CNA Waiing on PODs to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f NAME` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    if [[ $retVal -gt 5 ]] ; then
      echo "Failed POD Annotations test on $i. Failing Fast!"
      (( passed++ ))
      exit $retVal
    fi
  done
  echo "Test Passed for all PODs in $NEW_APP."
}

# TestRail Test Case C7441
createDockerPull() {
  echo "Test: Deploying with Docker Pull then oc new-app"
  docker pull alpine
  oc new-project tst-docker-pull
  oc new-app docker.io/alpine:latest
  i=0
  until oc get pods | grep -i alpine | cut -d ' ' -f 1 ; do
    sleep 2;
    (( i++ ))
  done
  output=$(oc get pods | grep -i alpine | cut -d ' ' -f 1 | sed 's/:.*//')
  if [ -z $output ] ; then
    echo "ERROR: No POD(s) found matching $output! - Exiting!"
    return 1;
  else
    echo "POD(s) $output found, w00t! Moving on..."
  fi
  x=0
  for my_pod in ${output[@]} ; do
    echo $my_pod;
    tstAnnotate $my_pod
    echo "Function exit was $?"
    if $? -gt 0 ; then
      (( x++ ))
    fi
  done
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
  i=0
  until oc get pods | grep -i pushtest | cut -d ' ' -f 1 ; do
    sleep 2;
    (( i++ ))
  done
  output=$(oc get pods | grep -i pushtest | cut -d ' ' -f 1 | sed 's/:.*//')
  if [ -z $output ] ; then
    echo "ERROR: No POD(s) found matching $output! - Exiting!"
    return 1;
  else
    echo "POD(s) $output found, w00t! Moving on..."
  fi
  x=0
}

# Test rail Test Case C7445
createS2i() {
  echo "Test: Deploy a Source to Image (S2i)"
  oc new-project puma-test-app
  oc new-app https://github.com/openshift/sti-ruby.git \
  --context-dir=2.0/test/puma-test-app
  i=0
  until oc get pods | grep -i sti-ruby | cut -d ' ' -f 1 ; do
    sleep 2;
    (( i++ ))
  done
  output=$(oc get pods | grep -i sti-ruby | cut -d ' ' -f 1 | sed 's/:.*//')
  if [ -z $output ] ; then
    echo "ERROR: No POD(s) found matching $output! - Exiting!"
    return 1;
  else
    echo "POD(s) $output found, w00t! Moving on..."
  fi
  x=0
  for my_pod in ${output[@]} ; do
    echo $my_pod;
    tstAnnotate $my_pod
    echo "Function exit was $?"
    if $? -gt 0 ; then
      (( x++ ))
    fi
  done
}

# Test Rail Test Case C7448
createTemplate() {
  echo "Test: Deploy an Image via OpenShift Template"
  oc new-project php
  oc new-app -f /usr/share/openshift/examples/quickstart-templates/rails-postgresql.json
  i=0
  until oc get pods | grep -i rails | cut -d ' ' -f 1 ; do
    sleep 2;
    (( i++ ))
  done
  output=$(oc get pods | grep -i rails | cut -d ' ' -f 1 | sed 's/:.*//')
  if [ -z $output ] ; then
    echo "ERROR: No POD(s) found matching $output! - Exiting!"
    return 1;
  else
    echo "POD(s) $output found, w00t! Moving on..."
  fi
  x=0
  for i in "${output[@]}" ; do
    echo $i | awk 'BEGIN{RS=" "} {print}';
    tstAnnotate $i
    echo "Function exit was $?"
    if $? -gt 0 ; then
      (( x++ ))
    fi
  done
}

# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Verify POD has been annotated with "BlackDuck"
tstAnnotate() {
  my_pod=$1
  echo "Now testing POD Annoations on: $my_pod"
  echo "Checking for BlackDuck POD annotations..."
  a_state=$(oc describe pod $my_pod | grep -i BlackDuck)
  echo "a_state"
  if [[ $a_state == "" ]]; then
    echo "ERROR: There appears to be no POD Annoations present on $my_pod!"
    exit 1;
  else
    echo "BlackDuck OpsSight Annoations found on $my_pod! TEST PASS"
  fi
}

burnItDown() {
  #Burn all the deployments down
  while oc get project | grep -q $NS ; do
    echo "`date` [[ burnItDown ]] still waiting `oc delete project $NS`"
    sleep 3
  done
}
createNs
createPod
if [[ $? -gt 0 ]]; then
  echo "failed @ $createPod"
  exit $?
fi

createDockerHub
if [[ $? -gt 0 ]]; then
  echo "failed @ $createDockerHub"
  exit $?
fi

createDockerPull
if [[ $? -gt 0 ]]; then
  echo "failed @ $createDockerPull"
  exit $?
fi

createDockerLoad
if [[ $? -gt 0 ]]; then
  echo "failed @ $createDockerLoad"
  exit $?
fi

createS2i
if [[ $? -gt 0 ]]; then
  echo "failed @ $createS2i"
  exit $?
fi

createTemplate
if [[ $? -gt 0 ]]; then
  echo "failed @ $createTemplate"
  exit $?
fi

burnItDown
# TODO Test Results:  Ran Pass Fail (from $x?)
# TEST RAN
createNewApp centos/python-35-centos7~https://github.com/openshift/django-ex.git
