FROM golang:1.8

ENV GOPATH=/go/
ENV GOBIN=/go/bin/
WORKDIR /

COPY ./ /go/src/bitbucket.org/bdsengineering/perceptor/

WORKDIR /go/src/github.com/blackducksoftware/canary/

RUN go install ./cmd/perceptor.go

CMD /go/bin/service_scanner
