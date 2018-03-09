#!/bin/bash
# Copyright 2018 Synopsys Inc.
# Author:  Joel (Shepp) Sheppard

## This file is still WIP

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


# Test Rail Test Case C7439
# This deployment type doesn't create a POD or PODs
# Only an ImageStream is created. - Removed from POD annoations test
# Tested working March 1, 2018
createDockerLoad() {
  set -e
  REGISTRY_PORT=":5000"
  NS=pushtest
  # NEW_APP=$1
  PODS=$2
  my_pod=$1
  echo "Test: Deploying via Docker Load..."
  echo "Pulling hello-world..."
  sudo docker pull hello-world
  echo "Saving hello-world as a tarball..."
  sudo docker save hello-world > /var/tmp/hello-world.tar
  if [[ $? -gt 0 ]] ; then
    echo "`ls -l /var/tmp` Cannot save hello-world as a tarball to /var/tmp exiting- Fail fast!"
    exit "$?"
  else
    echo "Yea, docker image saved successfully!!"
  fi
  echo "Loading image via docker load..."
  sudo docker load -i var/tmp/hello-world.tar
  if [[ $? -gt 0 ]] ; then
    echo "Image NOT Loaded, exiting - fail fast!!"
    exit "$?"
  else
    echo "Docker Load completed successfully!!! `sudo docker images | grep hello-world`"
  fi
  # Login to Openshift
  # TODO Fix this stupid shit below with the password in this file @JOEL!!
  oc login -u=clustadm -p=devops123!
  oc_token=$(oc whoami -t)
  echo "oc_token is: $oc_token"
  if [[ -z $oc_token ]] ; then
    echo "No oc-token found, exiting!!"
    exit 28
  else
    echo "oc_token found: $oc_token"
  fi
  # Swith to the default project (the registry is here)
  echo "Switching to project default, the docker registry lives there..."
  oc project default
  # Let's find the Openshift Registry IP
  # Field 5 is the IP, and we can assume the PORT will always be 5000 and export it
  echo "Finding the OpenShift Registry IP..."
  regIpPort=$(oc get svc | grep docker-registry | cut -d ' ' -f 5)$REGISTRY_PORT
  echo "Registry IP is: $regIpPort"
  if [[ -z $regIpPort ]] ; then
    echo "Something's wrong, cannot get the docker-registry IP and Port!!!"                                                                           │    pushtest
    exit 27
  else
    echo "Found the docker-registry and port: $regIpPort"
  fi
  # Now let's login to the Image Registry
  sudo docker login -u clustadm -e test@synopsys.com -p $oc_token $regIpPort
  # Create a project to push to
  # First let's see if the projects exists, nuke it if so
  if [[ -z $NS ]] ; then
    echo "Sweet, NS: $NS not found, let's do this!"
  else
    echo "Dang it!  NS: $NS Found, Frenzy needs to destroy it!!"
    burnItDown $NS
  fi
  echo "Creating new project '$NS'"
  oc new-project $NS
  # Now let's tag the image
  sudo docker tag docker.io/hello-world $regIpPort/pushtest
  if [[ $? -gt 0 ]] ; then
    echo "Tagging image has failed, exiting - FAIL FAST!"
    exit $?
  else
    echo "Docker image tagged successfully!"
  fi
  # Now push this puppy to the Registry
  echo "Pusing docker image to the OpenShift Registry..."
  sudo docker push $regIpPort/pushtest/hello-world
  if [[ $? -gt 0 ]] ; then
    echo "Docker push failed, exiting - FAIL FAST!"
    exit $?
  else
    echo "Docker push successful!"
  fi
  # Let's see if the pushtest ImageStream was created...
  echo "`oc get is` Did the deployment create and ImageStream?"
  echo "If there's no PODs, GTFO of the test!!"
  if [[ `oc get pods | grep -v STATUS | wc -l` -eq 0 ]] ; then
    echo "There are NO PODS for this deployment type [dockerLoad] - EXITING!!"
    exit 1
  else
    echo "PODs found, now we can move on to POD annoations tests with confidence."
  fi
  # Check PODs present
  until [[ `oc get pods | grep -v STATUS | wc -l` -ge "$PODS" ]] ; do
    echo "createDockerLoad: Waiing on POD(s) to come up, so far: `oc get pods | grep -v NAME`"
    sleep 3
  done
  echo "Done waiting!"
  for i in `oc get pods | grep -v build | grep -v deploy | grep -v NAME | cut -d ' ' -f 1` ; do
    tstAnnotate $i
    retVal=$?
    passed=0
    failed=0
    if [[ retVal -gt 0 ]] ; then
      echo "FAIL: createDockerLoad Failed POD Annoations test on $i.  Failing Fast!"
      (( failed++ ))
    else
      echo "Passed POD Annoations test on $i!"
      (( passed++ ))
    fi
  done
  echo "createDockerLoad Test Passed for all PODs in $NEW_APP."
  # Tear it down
  burnItDown $NS
}

# Verify POD has been annotated with "BlackDuck"
tstAnnotate() {
  my_pod=$1
  echo "Now testing POD Annoations on: $my_pod"
  echo "Checking for BlackDuck POD annotations..."
  a_state=$(oc describe pod $my_pod | grep -i BlackDuck)
  echo "a_state"
  if [[ $a_state -eq "" ]] ; then
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

createDockerLoad
#if [[ $? -gt 0 ]]; then
#  echo "failed @ $createDockerLoad"
#  exit $?
#fi
