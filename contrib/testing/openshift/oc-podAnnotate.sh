#!/bin/bash
# Copyright (C) 2018 Synopsys, Inc.
# Author Joel (Shepp) Sheppard

export REGISTRY_PORT=":5000"

sanity_check() {
  oc get version
  oc get pods
  if [[ "$?" == 0 ]] ; then
    echo "OpenShift seems okay."
  else
    echo "FAIL: The OpenShift preconditions for the oc client were not met, Exiting..."
    exit 22
  fi
}

# Create a Namespace for createPod test
# Tested working Feb 2018
createNs() {
  # Clean up NS JIC it's still here...
  oc get ns | grep $NS | cut -d ' ' -f 1 | xargs oc delete ns
  NS=perceptortestns
  PODS=$2
  echo "Creating Namespace..."
  oc create -f ./perceptorTestNS.yml
  until [[ `oc get ns | grep $NS | wc -l` -gt 0 ]] ; do
    echo "Waiting for Namespace to be created: `oc get ns | grep $NS`"
    sleep 3
  done
  echo "Done waiting!"
}
# Spin up a POD using oc run busybox
# TestRail Test Case C7556
# Deploy an image using OC RUN
# Tested as working Feb 2018
createPod() {
  NS=$NS
  PODS=$2
  if [[ "$PODS" -eq "" ]] ; then
    echo "ERROR: NO PODS FOUND, EXITING!!!"
    exit 23
  fi
  # my_pods=$1
  echo "Test: Creating POD using 'oc run'..."
  oc run busybox --image=busybox --namespace=$NS
  oc project $NS
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]]; do
    echo "createPod: Wating for POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
for i in `oc get pods | grep -v build | grep -v deploy | cut -d ' ' -f 1` ; do
  tstAnnotate $i
  retVal=$?
  passed=0
  failed=0
  if [[ retVal -gt 0 ]] ; then
    echo " FAIL: createPod Failed Annotations test on $i. Failing Fast!"
    (( failed++))
  else
    echo "Passed POD Annoations test on $i"
    (( passed++ ))
  fi
done
}

# TestRail Test Case C7440
# Tested as working Feb 2018
createDockerHub() {
  NS=tst-deploy-dockerhub
  NEW_APP=$1
  PODS=$2
  if [[ "$PODS" -eq "" ]] ; then
    echo "ERROR: NO PODS FOUND, EXITING!!!"
    exit 24
  fi
  my_pod=$1
  echo "Test: Deploying directly via DockerHUB with $NEW_APP and we want to see $PODS"
  oc new-project $NS
  oc new-app $NEW_APP
  echo "PODS equals: $PODS"
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "createDockerHub: Waiing on PODs to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=$?
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
      exit $retVal
    else
      echo "Passed POD Annoations test on $i!"
      (( passed++ ))
    fi
  done
  echo "createDockerHub Test Passed for all PODs in $NEW_APP."
}

# TestRail Test Case C7441
# Tested as working Feb 2018

createDockerPull() {
  NS=tst-docker-pull
  NEW_APP=$1
  PODS=$2
  if [[ "$PODS" -eq "" ]] ; then
    echo "ERROR: NO PODS FOUND, EXITING!!!"
    exit 25
  fi
  my_pod=$1
  echo "Test: Deploying with Docker Pull then oc new-app"
  docker pull alpine
  oc new-project $NS
  oc new-app $NEW_APP
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "createDockerPull: Waiing on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=$?
    passed=0
    failed=0
    if [[ retVal -gt 0 ]] ; then
      echo "FAIL: createDockerPull Failed POD Annoations test on $i.  Failing Fast!"
      (( failed++ ))
    else
      echo "Passed POD Annoations test on $i!"
      (( passed++ ))
    fi
  done
  echo "createDockerPull Test Passed for all PODs in $NEW_APP."
}

# Test rail Test Case C7445
# Creates a new app using Source to Image (S2i), a Builder and Application
# POD are created in this deployment.
# Tested as working March 1, 2018
createS2i() {
  NS=puma-test-app
  NEW_APP=$1
  PODS=$2
  my_pod=$1
  # Create a project to push to
  # First let's see if the project exists, nuke it if so
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found, let's do this!"
  else
    echo "Dang it!  NS: $NS Found, Frenzy needs to destroy it!!"
    burnItDown $NS
  fi
  echo "Test: Deploy a Source to Image (S2i)"
  oc new-project $NS
  oc new-app $NEW_APP
  # This deployment makes a few containers and it takes ~30s to spin them all up
  echo "PODS var is: $PODS"
  sleep 30
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "createS2i: Waiing on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=$?
    passed=0
    failed=0
    if [[ retVal -gt 0 ]] ; then
      echo "FAIL: [createS2i] Failed POD Annoations test on $i.  Failing Fast!"
      (( failed++ ))
    else
      echo "Passed POD Annoations test on $i!"
      (( passed++ ))
    fi
  done
  echo "[createS2i] Test Passed for all PODs in $NEW_APP."
}

# Test Rail Test Case C7448
# In progress March 1 2018

createTemplate() {
  NS=php
  NEW_APP=$1
  PODS=$2
  my_pod=$1
  # Create a project to deploy to
  # First let's see if the project exists, nuke it if so
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found, let's do this!"
  else
    echo "Dang it!  NS: $NS Found, Frenzy needs to destroy it!!"
    burnItDown $NS
  fi
  echo "Test: Deploy an Image via OpenShift Template"
  oc new-project $NS
  oc new-app -f /usr/share/openshift/examples/quickstart-templates/rails-postgresql.json
  echo "PODS var is: $PODS"
  sleep 30
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "[createTemplate]: Waiing on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=$?
    passed=0
    failed=0
    if [[ retVal -gt 0 ]] ; then
      echo "FAIL: [createTemplate] Failed POD Annoations test on $i.  Failing Fast!"
      (( failed++ ))
    else
      echo "Passed POD Annoations test on $i!"
      (( passed++ ))
    fi
  done
  echo "[createTemplate] Test Passed for all PODs in $NEW_APP."
}

# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Verify POD has been annotated with "BlackDuck"
tstAnnotate() {
  my_pod=$1
  echo "Now testing POD Annoations on: $my_pod"
  echo "Checking for BlackDuck POD annotations..."
  a_state=$(oc describe pod $my_pod | grep -i BlackDuck)
  echo "$a_state"
  if [[ -z $a_state ]] ; then
    echo "ERROR: There appears to be no POD Annoations present on $my_pod!"
    exit 1;
  else
    echo "BlackDuck OpsSight Annoations found on $my_pod! TEST PASS"
  fi
}

burnItDown() {
  #Burn all the deployments down
  echo "[burnItDown] START!!!"
  while oc get project | grep -i -q $NS ; do
    echo "`date` [[ burnItDown ]] still waiting `oc delete project $NS`"
    sleep 8
  done
  echo "[burnItDown] DONE!!"
}
createNs
createPod
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createPod"
#  exit $?
#fi

createDockerHub centos/python-35-centos7~https://github.com/openshift/django-ex.git
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createDockerHub"
#  exit $?
#fi

createDockerPull docker.io/alpine:latest 1
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createDockerPull"
#  exit $?
#fi

createS2i
#if [[ $? -gt 0 ]] ; then
#  echo "failed @ $createS2i"
#  exit $?
#fi

#createTemplate
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createTemplate"
#  exit $?
#fi

burnItDown
# TODO Test Results:  Ran Pass Fail (from $x?)
# TEST RAN
createNewApp centos/python-35-centos7~https://github.com/openshift/django-ex.git 1
