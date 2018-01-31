.PHONY: clean test compile build push
.DEFAULT_GOAL := compile

push:
	docker push registry.kipp.blackducksoftware.com/kube-perceptor-scanner:latest

build:
	docker build -t registry.kipp.blackducksoftware.com/kube-perceptor-scanner .

compile:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./dependencies/perceptor-scanner perceptor-scanner.go

test:
	go test ./...

clean:
	rm -rf ./dependencies/perceptor-scanner
