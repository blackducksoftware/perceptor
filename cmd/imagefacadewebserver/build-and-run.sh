set -e

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./imagefacadewebserver imagefacadewebserver.go

docker build -t mfenwickbd/imagefacadewebserver .
docker push mfenwickbd/imagefacadewebserver:latest


oc create -f if-serviceaccount.yaml --namespace=bds-perceptor
# allows launching of privileged containers for Docker machine access
oc adm policy add-scc-to-user privileged system:serviceaccount:bds-perceptor:if-sa
oc create -f if.yaml --namespace=bds-perceptor
oc create -f route.yaml --namespace=bds-perceptor
