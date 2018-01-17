set -e

# build kube-perceiver
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../cmd/kube-perceiver/kube-perceiver ../cmd/kube-perceiver/kube-perceiver.go
docker build -t mfenwickbd/kube-perceiver ../cmd/kube-perceiver/
docker push mfenwickbd/kube-perceiver:latest


# build perceptor
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../cmd/perceptor/dependencies/perceptor ../cmd/perceptor/perceptor.go
docker build -t mfenwickbd/perceptor ../cmd/perceptor/
docker push mfenwickbd/perceptor:latest
