#!/usr/bin/env bash
# TODO We can deploy this from a dockerfile, as a binary, once we have everything we want in it.

export PERCEPTOR=$GOPATH/src/bitbucket.org/bdsengineering/perceptor/

if [ ! -d $PERCEPTOR ]; then
	echo "Exiting the build: looks like your GOPATH isn't set up to have $PERCEPTOR"
	exit 1
fi

set -x

rm main

# This will add the 'perceptor' binary to your GOPATH.
rm $GOPATH/bin/perceptor
export GOBIN=$GOPATH/bin
go install ./cmd/perceptor/perceptor.go

# TODO this requires an `oc login` before it will work
#   or maybe a kubernetes login or something

KUBE_CONFIG=~/.kube/config
$GOPATH/bin/perceptor --kubeconfig=$KUBE_CONFIG --master=https://34.227.56.110.xip.io:8443
