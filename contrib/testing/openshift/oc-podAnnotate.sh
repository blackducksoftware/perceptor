#!/bin/bash
# Copyright (C) 2018 Synopsys, Inc.
# Author Joel (Shepp) Sheppard

export REGISTRY_PORT=":5000"

# Check the system is ready for testing...
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
  if [[ -z "$PODS" ]] ; then
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
  if [[ -z "$PODS" ]] ; then
    echo "ERROR: NO PODS FOUND, EXITING!!!"
    exit 24
  fi
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
  #NEW_APP=$1
  PODS=$2
  if [[ -z "$PODS" ]] ; then
    echo "ERROR: NO PODS FOUND, EXITING!!!"
    exit 25
  fi
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
  echo "createDockerPull Test Passed for all PODs!"
}

# Test rail Test Case C7445
# Creates a new app using Source to Image (S2i), a Builder and Application
# POD are created in this deployment.
# Tested as working March 1, 2018
createS2i() {
  NS=puma-test-app
  # NEW_APP=$1
  PODS=$2
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
  oc new-app https://github.com/openshift/sti-ruby.git --context-dir=2.0/test/puma-test-app
  # This deployment makes a few containers and it takes ~30s to spin them all up
  echo "PODS var is: $PODS"
  sleep 30
  # TODO Watch this below until, not sure its really doing what we want...  see createTemplate function
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
  #NEW_APP=$1
  PODS=$1z $
  # Create the project to deploy to
  # First let's see if the project exists, nuke it if so
  if [[ -z $NS ]] ; then
      echo "Sweet, NS: $NS not found, let's do this!"
  else
      echo "Dang it!  NS: $NS Found, Frenzy needs to destroy it!!"
      burnItDown $NS
  fi
  echo "Test: Deploy Image via OpenShift Template"
  oc new-project $NS
  oc new-app -f /usr/share/openshift/examples/quickstart-templates/rails-postgresql.json
  # Wait until the hook-pre POD has Completed before moving on to testing...
  until [[ `oc get pods | grep rails-postgresql-example-1-hook-pre | grep Completed` ]] ; do
        echo "[createTemplate]: Waiting on PODs to come up, so far: `oc get pods | grep -v NAME`"
        sleep 1
  done
  # After the rails-postgresql-example-1-hook-pre container Completes
  # We have to wait/sleep a little bit more for the last container to come up (~10s)
  # E.g. rails-postgresql-example-1-<random#> - Not ideal I know
  sleep 10
  echo "Waiting for the REAL rails-postgresql-1-<random> to come up..."
  echo -e "`oc get pods` \nIf we got here, then: Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | grep Running | cut -d ' ' -f 1` ; do
      # echo "[[DEBUGGIUNG]]: DOLLAR-EYE is: $i"
      tstAnnotate $i
      retVal=$?
      passed=0
      failed=0
      if [[ retVal -gt 0 ]] ; then
          echo "FAIL: [createTemplate] Failed POD Annotations Test on $i.  Failing Fast!"
          exit 46
          # echo "[[DEBUGGING]]: "$?""
          (( failed++ ))
      else
          echo "Passed POD Annotations test on $i!"
          (( passed++ ))
      fi
  done
  echo "[createTemplate] Test Passed for all PODs!"
}

# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Verify POD has been annotated with "BlackDuck"
tstAnnotate() {
  PODS=$1
  echo "New testing POD Annotations on: $PODS"
  echo "Checking for BlackDuck POD Annotations..."
  a_state=$(oc describe pod $PODS | grep BlackDuck)
  echo "$a_state"
  if [[ -z $a_state ]] ; then
      echo "ERROR: There appears to be no POD Annotations present on $PODS"
      exit 1;
  else
      echo "PASS: BlackDuck OpsSight POD Annotations found on POD: $PODS!  TEST PASS"
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

# TestRail Test Case C7556
createPod
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createPod"
#  exit $?
#fi

# TestRail Test Case C7440
createDockerHub centos/python-35-centos7~https://github.com/openshift/django-ex.git
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createDockerHub"
#  exit $?
#fi

# TestRail Test Case C7441
createDockerPull docker.io/alpine:latest 1
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createDockerPull"
#  exit $?
#fi

# Test rail Test Case C7445
createS2i
#if [[ $? -gt 0 ]] ; then
#  echo "failed @ $createS2i"
#  exit $?
#fi

# Test Rail Test Case C7448
createTemplate
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createTemplate"
#  exit $?
#fi

burnItDown
# TODO Test Results:  Ran Pass Fail (from $x?)
# TEST RAN
createNewApp centos/python-35-centos7~https://github.com/openshift/django-ex.git 1
