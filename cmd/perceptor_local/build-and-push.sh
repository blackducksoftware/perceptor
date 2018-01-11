set -e

cp ~/.kube/config ./dependencies/kubeconfig
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dependencies/perceptor_local ./perceptor_local.go

docker build -t mfenwickbd/perceptor_local .

docker push mfenwickbd/perceptor_local:latest
