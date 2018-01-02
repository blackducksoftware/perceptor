#!/usr/bin/env bash

export PERCEPTOR=$GOPATH/src/bitbucket.org/bdsengineering/perceptor/

if [ ! -d $PERCEPTOR ]; then
	echo "Exiting the build: looks like your GOPATH isn't set up to have $PERCEPTOR"
	exit 1
fi

set -x

rm $GOPATH/bin/perceptor
export GOBIN=$GOPATH/bin
CGO_ENABLED=0 GOOS=linux go install -a -tags netgo -ldflags '-w' ./cmd/perceptor/perceptor.go
# go install ./cmd/perceptor/perceptor.go
