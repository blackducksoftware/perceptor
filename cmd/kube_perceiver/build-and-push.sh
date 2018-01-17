set -e

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./kube-perceiver kube-perceiver.go

docker build -t mfenwickbd/kube-perceiver .

docker push mfenwickbd/kube-perceiver:latest
