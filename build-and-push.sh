set -e

cp ~/.kube/config ./dependencies/kubeconfig
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dependencies/perceptor ./cmd/perceptor/perceptor.go

docker build -t mfenwickbd/perceptor .

docker push mfenwickbd/perceptor:latest
