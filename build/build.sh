set -e

# build perceptor
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../cmd/perceptor/perceptor ../cmd/perceptor/perceptor.go
docker build -t mfenwickbd/perceptor ../cmd/perceptor/
docker push mfenwickbd/perceptor:latest


# build perceptor-scanner
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../cmd/perceptor-scanner/dependencies/perceptor-scanner ../cmd/perceptor-scanner/perceptor-scanner.go
docker build -t mfenwickbd/perceptor-scanner ../cmd/perceptor-scanner/
docker push mfenwickbd/perceptor-scanner:latest
