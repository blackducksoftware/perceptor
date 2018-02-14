set -e

cp ~/.kube/config ./dependencies/kubeconfig
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dependencies/perceptor_local ./perceptor_local.go

docker build -t mfenwickbd/perceptor_local .

# if running locally, can hit this from
#   http://localhost:3060/model
# docker run -p 3060:3001 mfenwickbd/perceptor_local
docker run -v /var/run/docker.sock:/var/run/docker.sock -p3060:3001 mfenwickbd/perceptor_local

# use one of these to just get a running container to play around in:
# docker run -ti mfenwickbd/perceptor_local sh
# docker run -v /var/run/docker.sock:/var/run/docker.sock -ti mfenwickbd/perceptor_local sh
