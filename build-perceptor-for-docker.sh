# env GOOS=linux GOARCH=386 go build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dependencies/perceptor ./cmd/perceptor/perceptor.go

docker build -t mfenwickbd/perceptor .

# if running locally, can hit this from
#   http://localhost:3060/model
docker run -p 3060:3000 mfenwickbd/perceptor
# docker run -v /var/run/docker.sock:/var/run/docker.sock -p3060:3000 mfenwickbd/perceptor

# use one of these to just get a running container to play around in:
# docker run -ti mfenwickbd/perceptor sh
# docker run -v /var/run/docker.sock:/var/run/docker.sock -ti mfenwickbd/perceptor sh

# To run on openshift:
# docker push mfenwickbd/perceptor:latest
# oc login -u admin -p 123
## preconditions: have deleted any existing app and images
# oc project perceptor-proj
# oc new-app mfenwickbd/perceptor
