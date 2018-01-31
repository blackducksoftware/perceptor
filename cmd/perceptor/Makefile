.PHONY: clean test compile build push
.DEFAULT_GOAL := compile

push:
	docker push registry.kipp.blackducksoftware.com/kube-perceptor:latest

build:
	docker build -t registry.kipp.blackducksoftware.com/kube-perceptor .

compile:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./dependencies/perceptor perceptor.go

test:
	go test ./...

clean:
	rm -rf ./dependencies/perceptor
