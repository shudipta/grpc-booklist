#!/usr/bin/env bash
set -xe

export REPO_ROOT=$GOPATH/src/github.com/shudipta/grpc-booklist

pushd $REPO_ROOT
    # build binary
    go build -o hack/docker/grpc-booklist server/server.go
    # build docker image
    docker build -t shudipta/grpc-booklist:latest hack/docker/
    docker push shudipta/grpc-booklist:latest
    # load docker image to minikube
    docker save shudipta/grpc-booklist:latest | pv | (eval $(minikube docker-env) && docker load)
    # deploy in kubernetes
    kubectl delete deploy -l app=grpc
    kubectl delete svc -l app=grpc
    kubectl apply -f hack/deploy/grpc-booklist.yaml
    # clean
    rm -rf hack/docker/grpc-booklist
popd


# # create necessary TLS certificates
# ./onessl create ca-cert
# ./onessl create server-cert server --domains=foo-operator.default.svc
# export SERVICE_SERVING_CERT_CA=$(cat ca.crt | ./onessl base64)
# export TLS_SERVING_CERT=$(cat server.crt | ./onessl base64)
# export TLS_SERVING_KEY=$(cat server.key | ./onessl base64)
# export KUBE_CA=$(./onessl get kube-ca | ./onessl base64)

# # create operator deployment, service and tls secret
# cat operator.yaml | ./onessl envsubst | kubectl apply -f -

# # create APIService
# cat apiservice.yaml | ./onessl envsubst | kubectl apply -f -

# # cleanup
# rm foo ca.crt ca.key server.crt server.key