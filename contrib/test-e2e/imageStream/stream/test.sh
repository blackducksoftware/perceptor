#!/bin/bash

expectedIs=("nats" "hello-world")
expectedIsTags=("nats:1.0.2" "nats:1.0.4" "hello-world:latest")
expectedSHA=("4b0edace2daf8dfd6d7b93addff9eabd4a89a398c51a9930263fb9e0ad938d37" \
"61fcb1f40da2111434fc910b0865c54155cd6e5f7c42e56e031c3f35a9998075" \
"66ef312bbac49c39a89aa9bcc3cb4f3c9e7de3788c944158df3ee0176d32b751")
command=$1
project=$2

# Skip the test if it is not OpenShift
if [[ $command -ne 'oc' ]]; then
  exit 0
fi

# Verify that the ImageStreams are not existed
initialVerification() {
  # Check each imagestream in the pre-defined list
  for i in "${expectedIs[@]}";
  do
    found=$(oc get is $i  -n $project | grep $i | wc -l)
    if [[ $found -eq 1 ]]; then
      echo "Image Stream $i already exists! Please delete the ImageStream to proceed!"
      exit 1
    fi
  done
}

initialVerification

cat << EOF > imgstream.yml
kind: ImageStreamList
apiVersion: v1
metadata: {}
items:
# - kind: ImageStream
#   apiVersion: v1
#   metadata:
#     name: alpine
#   spec:
#     tags:
#     - name: '3.1'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.1
#     - name: '3.2'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.2
#     - name: '3.3'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.3
#     - name: '3.4'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.4
#     - name: '3.5'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.5
#     - name: '3.6'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.6
#     - name: '3.7'
#       from:
#         kind: DockerImage
#         name: docker.io/alpine:3.7
# - kind: ImageStream
#   apiVersion: v1
#   metadata:
#     name: busybox
#   spec:
#     tags:
#     - name: '1.23'
#       from:
#         kind: DockerImage
#         name: docker.io/busybox:1.23
#     - name: '1.24'
#       from:
#         kind: DockerImage
#         name: docker.io/busybox:1.24
#     - name: '1.25'
#       from:
#         kind: DockerImage
#         name: docker.io/busybox:1.25
#     - name: '1.26'
#       from:
#         kind: DockerImage
#         name: docker.io/busybox:1.26
#     - name: '1.27'
#       from:
#         kind: DockerImage
#         name: docker.io/busybox:1.27
#     - name: '1.28'
#       from:
#         kind: DockerImage
#         name: docker.io/busybox:1.28
- kind: ImageStream
  apiVersion: v1
  metadata:
    name: hello-world
  spec:
    tags:
    - name: latest
      from:
        kind: DockerImage
        name: docker.io/hello-world@sha256:66ef312bbac49c39a89aa9bcc3cb4f3c9e7de3788c944158df3ee0176d32b751
- kind: ImageStream
  apiVersion: v1
  metadata:
    name: nats
  spec:
    tags:
    # - name: '0.6.4'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:b52d9059a37ccf8340d683378f55dc988247be21ed85910ba9a66ca4b5ac696d
    # - name: '0.6.6'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:865f4c9dec716b35499149cbac95fdf639b8f51c73e2e135372c478212ca465e
    # - name: '0.6.8'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:9cfe10092e5ea3faa7e83f26e53adc60b6aacce1215ea8d573c2fb006c4c0713
    # - name: '0.7.2'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:08e66273ba601bd7b0a791d2ff1fc4f8f1c8be82128126869c63261795318683
    # - name: '0.8.0'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:2dfb204c4d8ca4391dbe25028099535745b3a73d0cf443ca20a7e2504ba93b26
    # - name: '0.8.1'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:3329c27c3e434febd0de986b5685e5fdcf1290e728b9a6c212cdf983ba3a4e41
    # - name: '0.9.2'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:34e912f2fe65f9133f9b2f76cee746bd1ac8dc7a096fc3a4183fcfa588e44972
    # - name: '0.9.4'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:b22e176b878c315daac41d7f8b8b23d9ce6d738e0938a73389406513af9713f1
    # - name: '0.9.6'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:47b825feb34e545317c4ad122bd1a752a3172bbbc72104fc7fb5e57cf90f79e4
    # - name: '1.0.0'
    #   from:
    #     kind: DockerImage
    #     name: docker.io/nats@sha256:35a8c43414d7fd8708f91189c36986e7a0ff0ef3c9bdd98b8df3809e2a2dc306
    - name: '1.0.2'
      from:
        kind: DockerImage
        name: docker.io/nats@sha256:4b0edace2daf8dfd6d7b93addff9eabd4a89a398c51a9930263fb9e0ad938d37
    - name: '1.0.4'
      from:
        kind: DockerImage
        name: docker.io/nats@sha256:61fcb1f40da2111434fc910b0865c54155cd6e5f7c42e56e031c3f35a9998075
EOF

#########
oc create -f imgstream.yml -n $project

pollAndVerifyImageStreamCreation() {

  arraylength=${#expectedIsTags[@]}
  found=0
  # Continue until all imagestreams are created
  until [ $arraylength == $found ] ;
  do
    # Check each imagestream in the pre-defined list
    for i in "${expectedIsTags[@]}";
    do
      imagestreams=($(echo "$i" | tr ':' '\n'))
      # Check each imagestream tag in the pre-defined list
      status=false
      until [ $status == true ] ;
      do
        for j in $(oc get is ${imagestreams[0]}  -n $project | awk '{print $3}' | tail -1 | tr "," "\n")
        do
          if [[ "${imagestreams[1]}" == "$j" ]]; then
            status=true
            echo "Found $i!"
            found=$(expr $found + 1)
          fi
        done
      done
    done
  done

  echo "$found imagestreams were found!"
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

pollAndVerifyImageStreamScan() {

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

  echo "$found imagestreams were scanned!"

}

pollAndVerifyImageStreamAnnotation () {
  # Check each new-app in the pre-defined list
  for i in "${expectedIs[@]}";
  do
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

pollAndVerifyImageStreamCreation
pollAndVerifyImageStreamScan
oc delete -f imgstream.yml -n $project
