#!/bin/bash
# Copyright (C) 2018 Synopsys, Inc.
# Authors:  Joel Sheppard, Jay Vyas

# Redirect output to a log and visible at the console
exec > >(tee /var/tmp/pod-annotations-tests.log) 2>&1

# Export the OpenShift internal Registry Port
export REGISTRY_PORT=":5000"

sanity_check() {
  oc login -u=clustadm -p=devops123!
  oc get version
  oc get pods
  if [[ "$?" == 0 ]] ; then
    echo "OpenShift seems good."
  else
    echo "[FAIL]: The OpenShift client preconditions have not been met, EXITING!"
    exit 22
  fi
}

# Create the Namespace for createPod test
createNs() {
  export POD_NS="perceptortestns"
  # Clean up NS JIC it's still here...
  sudo oc get ns | grep $POD_NS | cut -d ' ' -f 1 | xargs oc delete ns
  while [[ `sudo oc get ns | grep $POD_NS | wc -l` -gt 0 ]] ; do
    echo "[createNs]: Waiing on Frenzy to destroy NS: $NS `oc get ns | grep $NS`"
    sleep 3
  done
  echo "Done waiting!"
  oc create -f ./perceptorTestNS.yml
  while [[ `sudo oc get ns | grep $POD_NS | wc -l` -gt 0 ]] ; do
    echo "[createNs]: Waiing on NS to come up: `oc get ns | grep $NS`"
    sleep 3
  done
  ns_state=$(oc get ns | grep $POD_NS)
  if [ -z $ns_state ] ; then
    echo "ERROR: Namespace $POD_NS not found!"
    exit 1;
  else
    echo "Namespace $POD_NS found, w00t! Moving on..."
  fi
}
# Spin up a POD using oc run busybox
# TestRail Test Case C7556
# Deploy an image using OC RUN
createPod() {
  PODS=1
  echo "[Test]: Creating POD using 'oc run'..."
  oc run busybox --image=busybox --namespace=$POD_NS
  oc project perceptortestns
  my_pod=$(oc get pods | grep -i busybox | cut -d ' ' -f 1)
  if [ -z $my_pod ] ; then
    echo "ERROR: No POD found matching $my_pod!"
    return 1;
  else
    echo "POD name $my_pod found, w00t! Moving on..."
    echo "$my_pod"
  fi
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "[createPod]: Waiing on PODs to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "[ERROR]: [createPod] Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
    else
      echo "[createPod] Passed POD Annoations test on $i!"
      (( passed++))
    fi
  done
  echo "[createPod] Test Passed for all PODs!"
}

# TestRail Test Case C7440
# Creates a new app using Source to Image (S2i), a Builder and Applicatiob
# POD are created in this deployment.
createDockerHub() {
  NS=tst-deploy-dockerhub
  NEW_APP=centos/python-35-centos7~https://github.com/openshift/django-ex.git
  PODS=$2
  my_pod=$1
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found.  We can begin the test!"
  else
    echo "Dang it!  NS: $NS found, FRENZY needs to destroy the NS!!!"
    burnItDown $NS
  fi
  echo "[Test]: Deploying directly via DockerHUB with $NEW_APP and we want to see $PODS"
  oc new-project $NS
  oc new-app $NEW_APP
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "[createDockerHub]: Waiing on PODs to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "[ERROR]: [createDockerHub] Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
    else
      echo "[createDockerHub] Passed POD Annoations test on $i!"
      (( passed++))
    fi
  done
  echo "[createDockerHub] Test Passed for all PODs in $NEW_APP."
}

# TestRail Test Case C7441
createDockerPull() {
  NS=tst-docker-pull
  PODS=1
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found.  We can begin the test!"
  else
    echo "Dang it!  NS: $NS found, FRENZY needs to destroy the NS!!!"
    burnItDown $NS
  fi
  echo "Test: Deploying with Docker Pull then oc new-app"
  docker pull alpine
  oc new-project $NS
  oc new-app docker.io/alpine:latest
  until [[ `oc get pods | grep -v STATUS | grep -v build | grep -v deploy | grep Completed | wc -l` -ge "$PODS" ]] ; do
    echo "[createDockerPull]: Waiing on PODs to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  # Sleep 5s to let the POD finish coming up
  sleep 5
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "[ERROR]:[createDockerPull] Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
    else
      echo "[createDockerPull] Passed POD Annoations test on $i!"
      (( passed++ ))
    fi
  done
  echo "[createDockerPull] Test Passed for all PODs!"
}

# Test Rail Test Case C7439
createDockerLoad() {
  NS=pushtest
  PODS=$1
  echo "[Test]: Deploying via Docker Load..."
  echo "Pulling hello-world..."
  sudo docker pull hello-world
  echo "Saving hello-world as a tarball..."
  sudo docker save hello-world > /var/tmp/hello-world.tar
  if [[ "$?" -gt 0 ]] ; then
    echo "`ls -l /var/tmp/ | grep "hello-world.tar"` [FAIL]: Unable to save hello-world as a tarball to /var/tmp EXITING - Fail FAST!"
    exit "$?"
  else
    echo "YEA! Docker Image Loaded Successfully!"
  fi
  echo "Loading image via docker load..."
  sudo docker load -i /var/tmp/hello-world.tar
  if [[ "$?" -gt 0 ]] ; then
    echo "[ERROR]: Docker Image NOT Loaded! Exiting, Fail Fast!"
    exit 27
  else
    echo "Docker Load completed successfully!! `sudo docker images | grep "hello-world"`"
  fi
  # Login to Openshift
  # TODO Do something better for the UN and PW???
  echo "Logging into OpenShift..."
  oc login -u=clustadm -p=devops123!
  oc_token=$(oc whoami -t)
  echo "The Token is: $oc_token."
  if [[ -z $oc_token ]] ; then
    echo "[ERROR]: No oc_token found!  Exiting! Fail Fast!"
    exit 28
  else
    echo "oc_token found: $oc_token"
  fi
  # Swith to the default project (the internal registry is in this project)
  echo "Swtiching to the 'default' project, the Docker Registry lives here..."
  oc project default
  # Let's find the Openshift Registry IP
  # Field 5 is the IP, and we can assume the PORT will always be 5000 and export that
  echo "Finding the OpenShift Registry Internal IP..."
  regIpPort=$(oc get svc | grep docker-registry | cut -d ' ' -f 5)$REGISTRY_PORT
  echo "Registry IP and Port are: $regIpPort"
  if [[ -z $regIpPort ]] ; then
    echo "[ERROR]: Something's wrong, cannot get the docker-registry IP and Port!"
    exit 29
  else
    echo "Found the docker-registry IP and Port: $regIpPort"
  fi
  # Now let's login to the Image Registry
  echo "Logging into the Default Image Registry..."
  sudo docker login -u clustadm -e test@synopsys.com -p $oc_token $regIpPort
  # Create a project to push to
  # First, let's see if the project already exists and burnItDown it if present
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found.  We can begin the test!"
  else
    echo "Dang it!  NS: $NS found, FRENZY needs to destroy the NS!!!"
    burnItDown $NS
  fi
  echo "Creating new project: $NS"
  oc new-project $NS
  # Now let's tag the image
  echo "Tagging the pushtest Image..."
  sudo docker tag docker.io/hello-world $regIpPort/pushtest
  if [[ $? -gt 0 ]] ; then
    echo "[ERROR]: Tagging Image has failed, Exiting - Fail Fast!"
    exit $?
  else
    echo "Docker Image tagged successfully!"
  fi
  # Now push this puppy to the Registry
  echo "Pushing Docker Image to the OpenShift Registry..."
  sudo docker push $regIpPort/pushtest/hello-world
  if [[ $? -gt 0 ]] ; then
    echo "[ERROR]: Docker Push has failed, Exiting - Fail Fast!"
    exit $?
  else
    echo "Docker Push successful!"
  fi
  # Let's see if the pushtest POD is created...
  echo "Check that the POD has been created, if not GTFO!!"
  if [[ `oc get pods | grep -v STATUS | wc -l` -eq 0 ]] ; then
    echo "[ERROR]: There are no PODs present for this deployment type (dockerLoad), Exiting!"
    exit 30
  else
    echo "PODs Found!  Now we can proceed with confidence!"
  fi
  # Okay we've made it this far, let's now move forward with POD Annoations testing
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge $PODS ]] ; do
    echo "[createDockerLoad]: Waiting on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "[ERROR]: [createDockerLoad] Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
    else
      echo "[createDockerLoad] Passed POD Annoations test on $i!"
      (( passed++))
    fi
  done
  echo "[createDockerLoad] Test Passed for all PODs!"
}

# Test rail Test Case C7445
createS2i() {
  NS=puma-test-app
  # This deployment reults in a net 2 PODs
  PODS=2
  # Check the namespace exists and Frenzy it if so...
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found.  We can begin the test!"
  else
    echo "Dang it!  NS: $NS found, FRENZY needs to destroy the NS!!!"
    burnItDown $NS
  fi
  echo "Test: Deploy a Source to Image (S2i)"
  oc new-project $NS
  echo "Deploying the S2i App..."
  oc new-app https://github.com/openshift/sti-ruby.git \
  --context-dir=2.0/test/puma-test-app
  # Wait for PODs to come up
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge $PODS ]] ; do
    echo "[createS2i]: Waiting on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  # Sleep just 5s while the actual Ruby App POD comes up...
  sleep 5
  # Start passing in one POD at a time to check for annoations
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "[ERROR]: [createS2i] Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
    else
      echo "[createS2i] Passed POD Annoations test on $i!"
      (( passed++))
    fi
  done
  echo "[createS2i] Test Passed for all PODs in!"
}

# Test Rail Test Case C7448
createTemplate() {
  NS=php
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found.  We can begin the test!"
  else
    echo "Dang it!  NS: $NS found, FRENZY needs to destroy the NS!!!"
    burnItDown $NS
  fi
  echo "Test: Deploy an Image via OpenShift Template"
  oc new-project $NS
  oc new-app -f /usr/share/openshift/examples/quickstart-templates/rails-postgresql.json
  until [[ `oc get pods | grep "rails-postgresql-example-1-hook-pre"` ]] ; do
    echo "[createTemplate]: Waiting on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 2
  done
  echo "Done waiting!"
  # Now sleep 10s while the hook-pre finishes and brings up the final pod (rails-postgresql-example-1-<randomNum>)
  sleep 10
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=0
    passed=0
    failed=0
    if [[ $retVal -gt 0 ]] ; then
      echo "[ERROR]: [createTemplate] Failed POD Annotations test on $i. Failing Fast!"
      (( failed++ ))
    else
      echo "[createTemplate] Passed POD Annoations test on $i!"
      (( passed++))
    fi
  done
  echo "[createTemplate] Test Passed for all PODs!"
}

# TODO Verify perceptor is notified of new POD/Image - not sure how yet...

# Verify POD has been annotated with "BlackDuck"
tstAnnotate() {
my_pod=$1
retry=1
echo "Now testing POD Annoations on: $my_pod"
echo "Checking for BlackDuck POD annotations..."
a_state=$(oc describe pod $my_pod | grep -i BlackDuck)
until [[ $retry -gt 5 ]] ; do
  if [[ -z $a_state ]]; then
    echo "[ERROR]: [tstAnnotate] POD Annotations not found on $my_pod!"
    echo "Retrying... $retry"
    (( retry ++ ))
    sleep 3
  else
    echo "BlackDuck OpsSight Annoations found on $my_pod! TEST PASS"
    echo "Annoations: $a_state"
    break
  fi
done
}

burnItDown() {
  #Burn all the deployments down
  while oc get project | grep -q $NS ; do
    echo "`date` [[ burnItDown ]] still waiting `oc delete project $NS`"
    sleep 3
  done
}

sanity_check
createNs
createPod
if [[ $? -gt 0 ]]; then
  echo "failed @ createPod"
  exit $?
fi

createDockerHub
if [[ $? -gt 0 ]]; then
  echo "failed @ createDockerHub"
  exit $?
fi

createDockerPull
if [[ $? -gt 0 ]]; then
  echo "failed @ createDockerPull"
  exit $?
fi

#createDockerLoad
if [[ $? -gt 0 ]]; then
  echo "failed @ createDockerLoad"
  exit $?
fi

createS2i
if [[ $? -gt 0 ]]; then
  echo "failed @ createS2i"
  exit $?
fi

createTemplate
if [[ $? -gt 0 ]]; then
  echo "failed @ createTemplate"
  exit $?
fi

burnItDown
# TODO Test Results:  Ran Pass Fail (from $x?)
# TEST RAN
createNewApp centos/python-35-centos7~https://github.com/openshift/django-ex.git
