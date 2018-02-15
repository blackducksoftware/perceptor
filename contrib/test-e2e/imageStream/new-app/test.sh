#!/bin/bash

expected=("alpine-echoer-32")
expectedSHA=("13b626743aab93e7368a736127f31e6e028dc5da6cbf597ad45561404631abee")
command=$1
project=$2

# Skip the test if it is not OpenShift
if [[ $command != "oc" ]]; then
  exit 0
fi

# Verify that the Pods are not existed
initialVerification() {
  # Check each Pods in the pre-defined list
  for i in "${expected[@]}";
  do
    found=$(oc get dc -n $project | awk '{print $1}' | grep -xc $i)
    if [[ $found -eq 1 ]]; then
      echo "Deployment Config $i already exists! Please delete the Deployment Config to proceed!"
      exit 1
    fi

    found=$(oc get is -n $project | awk '{print $1}' | grep -xc $i)
    if [[ $found -eq 1 ]]; then
      echo "ImageStream $i already exists! Please delete the ImageStream to proceed!"
      exit 1
    fi
  done
}

initialVerification

oc new-app mfenwickbd/alpine-echoer-3.2@sha256:13b626743aab93e7368a736127f31e6e028dc5da6cbf597ad45561404631abee -n $project

pollAndVerifyNewAppCreation() {
  arraylength=${#expected[@]}
  found=0
  # Continue until all new-apps are created
  until [ $arraylength == $found ] ;
  do
    # Check each new-app in the pre-defined list
    for i in "${expected[@]}";
    do
      polls=0
      # Continue until the pods are created
      until [ $(oc get pods -n $project | grep Running | awk '{print $1}' | grep -c $i) = 1 ] ; do
        echo "waiting for $i to be up!"
        ((polls+=1))
        # Pod creation Exhausted. Check the cluster for the issue
        if [[ $polls -gt 48 ]] ; then
          echo "$i never came online! "
          exit 1
        fi
        sleep 5
      done

      polls=0
      # Continue until the imagestreams are created
      until [ $(oc get is -n $project | awk '{print $1}' | grep -xc $i) = 1 ] ; do
        echo "waiting for $i to be up!"
        ((polls+=1))
        # Pod creation Exhausted. Check the cluster for the issue
        if [[ $polls -gt 48 ]] ; then
          echo "$i never came online! "
          exit 1
        fi
        sleep 5
      done
      echo "Found $i!"
      found=$(expr $found + 1)
    done
  done

  echo "$found pods were found!"
}

return_scan_status ()
{
  if [ -z "$1" ] # Is parameter #1 zero length?
  then
    echo "-Parameter #1 is zero length." # Or no parameter passed.
  fi

  #executeCmd="wget -qO- http://10.24.2.145:3000/model | jq -r '.Images[\""$1"\"].ScanStatus'"
  executeCmd="wget -qO- http://perceptor:3001/model | jq -r '.Images[\""$1"\"].ScanStatus'"
  scanStatus=$(eval $executeCmd)
}

pollAndVerifyNewAppScan() {
  arraylength=${#expectedSHA[@]}
  found=0

  # Continue until all pods are scanned
  until [ $arraylength == $found ] ;
  do
    # Process each container in the pre-defined list
    for i in "${expectedSHA[@]}";
    do
      polls=0
      scanStatus=null
      return_scan_status $i
      # Continue until the scan status is ScanStatusCompleted
      until [ $scanStatus == "ScanStatusComplete" ] ; do
        echo "waiting for scan $i container to complete!"
        ((polls+=1))
        # Scan Exhausted. Check the hub
        if [[ $polls -gt 48 ]] ; then
          echo "Scan $i never completed! "
          exit 1
        fi
        sleep 5
      done
      echo "Scanned $i!"
      found=$(expr $found + 1)
    done
  done

  echo "$found new-app images were scanned!"
}

pollAndVerifyNewAppAnnotation () {
  # Check each new-app in the pre-defined list
  for i in "${expected[@]}";
  do
    podName=$(oc get pods -n $project | grep Running | awk '{print $1}' | grep $i)
    echo "Finding Annotations and Labels for pod: $podName"
    verifyAnnotation pod annotations $podName blackducksoftware.com/attestation-hub-server
    verifyAnnotation pod annotations $podName blackducksoftware.com/hub-scanner-version
    verifyAnnotation pod annotations $podName blackducksoftware.com/project-endpoint
    verifyAnnotation pod labels $podName com.blackducksoftware.image.has-policy-violations
    verifyAnnotation pod labels $podName com.blackducksoftware.image.has-vulnerabilities
    verifyAnnotation pod labels $podName com.blackducksoftware.image.policy-violations
    verifyAnnotation pod labels $podName com.blackducksoftware.image.vulnerabilities

    echo "Finding Annotations and Labels for ImageStream: $i"
    verifyAnnotation is annotations $i blackducksoftware.com/attestation-hub-server
    verifyAnnotation is annotations $i blackducksoftware.com/hub-scanner-version
    verifyAnnotation is annotations $i blackducksoftware.com/project-endpoint
    verifyAnnotation is labels $i com.blackducksoftware.image.has-policy-violations
    verifyAnnotation is labels $i com.blackducksoftware.image.has-vulnerabilities
    verifyAnnotation is labels $i com.blackducksoftware.image.policy-violations
    verifyAnnotation is labels $i com.blackducksoftware.image.vulnerabilities
  done
}

verifyAnnotation () {
  storageType=$1
  metadataType=$2
  name=$3
  annotationParam=$4
  executeCmd="oc get $1 $name -n $project -o json | jq '.metadata.$2.\"$4\"'"
  annotationVal=$(eval $executeCmd)
  if [[ $annotationVal != null ]]; then
    echo "$4 Annotation found for $3"
  else
    echo "$4 Annotation not found for $3"
    exit 1
  fi
}

deleteNewApp () {
  # Check each new-app in the pre-defined list
  for i in "${expected[@]}";
  do
    oc delete dc $i -n $project
    oc delete is $i -n $project
    oc delete svc $i -n $project
  done
}

pollAndVerifyNewAppCreation
pollAndVerifyNewAppScan
pollAndVerifyNewAppAnnotation
deleteNewApp
