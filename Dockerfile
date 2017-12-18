FROM golang:1.8

ENV GOPATH=/go/
ENV GOBIN=/go/bin/
WORKDIR /

COPY ./ /go/src/github.com/blackducksoftware/canary/

WORKDIR /go/src/github.com/blackducksoftware/canary/

RUN go install ./cmd/sidecar/service_scanner.go

CMD /go/bin/service_scanner
