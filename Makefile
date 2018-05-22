ifndef REGISTRY
REGISTRY=gcr.io/gke-verification
endif

ifdef IMAGE_PREFIX
PREFIX="$(IMAGE_PREFIX)-"
endif

ifneq (, $(findstring gcr.io,$(REGISTRY)))
PREFIX_CMD="gcloud"
DOCKER_OPTS="--"
endif

CURRENT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
OUTDIR=_output

.PHONY: test ${OUTDIR}

all: compile

compile:
	# Simple easy compile, completely docker driven.
	# Copy everything into an idiomatic gopath for easy debugging.  Build and copy to a build/perceptor binary.
	docker run -t -i --rm -v ${CURRENT_DIR}:/go/src/github.com/blackducksoftware/perceptor/ -w /go/src/github.com/blackducksoftware/perceptor/cmd/perceptor -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.9 go build -o perceptor
	cp cmd/perceptor/perceptor $(OUTDIR)

container: compile
	cd ${CURRENT_DIR}/cmd/perceptor; docker build -t $(REGISTRY)/$(PREFIX)perceptor .

push: container
	$(PREFIX_CMD) docker $(DOCKER_OPTS) push $(REGISTRY)/$(PREFIX)perceptor:latest

test:
	docker run -t -i --rm -v ${CURRENT_DIR}:/go/src/github.com/blackducksoftware/perceptor/ -w /go/src/github.com/blackducksoftware/perceptor -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.9 go test ./pkg/...

clean:
	rm -rf ${OUTDIR} cmd/perceptor/perceptor

${OUTDIR}:
	mkdir -p ${OUTDIR}

lint:
	./hack/verify-gofmt.sh
	./hack/verify-golint.sh
	./hack/verify-govet.sh
