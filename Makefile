.DEFAULT_GOAL := compile

compile:
	# Simple easy compile, completely docker driven.
	# Copy everything into an idiomatic gopath for easy debugging.  Build and copy to a build/perceptor binary.
	docker run -t -i --rm -v $(shell pwd):/go/src/github.com/blackducksoftware/perceptor/ -w /go/src/github.com/blackducksoftware/perceptor -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.9 go build -o build/perceptor ./cmd/perceptor

swagger:
	docker run --rm -it -v $(shell pwd):/go/src/github.com/blackducksoftware/perceptor -w /go/src/github.com/blackducksoftware/perceptor -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 quay.io/goswagger/swagger generate spec -m -b ./cmd/perceptor -o build/perceptor-swagger-spec.json

test:
	go test ./pkg/...
