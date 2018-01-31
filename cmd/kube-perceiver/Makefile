.PHONY: clean test compile build push
.DEFAULT_GOAL := compile

push:
	docker push registry.kipp.blackducksoftware.com/kube-perceiver:latest

build:
	docker build -t registry.kipp.blackducksoftware.com/kube-perceiver .

compile:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./dependencies/kube-perceiver kube-perceiver.go

test:
	go test ./...

clean:
	rm -rf ./kube-perceiver
