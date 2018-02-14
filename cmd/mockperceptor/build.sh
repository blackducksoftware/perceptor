set -e

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./mockperceptor mockperceptor.go

docker build -t mfenwickbd/mockperceptor .
docker push mfenwickbd/mockperceptor:latest
