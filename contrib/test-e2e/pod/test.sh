#!/bin/bash

expected=("alpine")
expectedSHA=("7df6db5aa61ae9480f52f0b3a06a140ab98d427f86d8d5de0bedab9b8df6b1c0")
command=$1
project=$2

# Verify that the Pods are not existed
initialVerification() {
  # Check each Pods in the pre-defined list
  for i in "${expected[@]}";
  do
    found=$($command get pod $i -n $project | grep $i | wc -l)
    if [[ $found -eq 1 ]]; then
      echo "Pod $i already exists! Please delete the Pod to proceed!"
      exit 1
    fi
  done
}

initialVerification

cat << EOF > pod.yml
kind: PodList
apiVersion: v1
metadata: {}
items:
- kind: Pod
  apiVersion: v1
  metadata:
    name: alpine
  spec:
    containers:
    - image: alpine@sha256:7df6db5aa61ae9480f52f0b3a06a140ab98d427f86d8d5de0bedab9b8df6b1c0
      command:
        - sleep
        - "3600"
      imagePullPolicy: IfNotPresent
      name: alpine
    restartPolicy: Always
EOF

#########
$command create -f pod.yml -n $project

pollAndVerifyPodCreation() {
  arraylength=${#expected[@]}
  found=0
  # Continue until all pods are created
  until [ $arraylength == $found ] ;
  do
    # Check each pod in the pre-defined list
    for i in "${expected[@]}";
    do
      polls=0
      # Continue until the pods are created
      until [ $($command get pods -n $project | grep Running | awk '{print $1}' | grep -xc $i) = 1 ] ; do
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

pollAndVerifyPodScan() {
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
      until [ "$scanStatus" == "ScanStatusComplete" ] ; do
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

  echo "$found pods were scanned!"
}

pollAndVerifyPodAnnotation () {
  # Check each new-app in the pre-defined list
  for i in "${expected[@]}";
  do
    verifyAnnotation pod annotations $i blackducksoftware.com/attestation-hub-server
    verifyAnnotation pod annotations $i blackducksoftware.com/hub-scanner-version
    verifyAnnotation pod annotations $i blackducksoftware.com/project-endpoint
    verifyAnnotation pod labels $i com.blackducksoftware.image.has-policy-violations
    verifyAnnotation pod labels $i com.blackducksoftware.image.has-vulnerabilities
    verifyAnnotation pod labels $i com.blackducksoftware.image.policy-violations
    verifyAnnotation pod labels $i com.blackducksoftware.image.vulnerabilities
  done
}

verifyAnnotation () {
  storageType=$1
  metadataType=$2
  name=$3
  annotationParam=$4
  executeCmd="$command get $1 $name -n $project -o json | jq '.metadata.$2.\"$4\"'"
  annotationVal=$(eval $executeCmd)
  if [[ $annotationVal != null ]]; then
    echo "$4 Annotation found for $3"
  else
    echo "$4 Annotation not found for $3"
    exit 1
  fi
}

pollAndVerifyPodCreation
pollAndVerifyPodScan
pollAndVerifyPodAnnotation
$command delete -f pod.yml -n $project
