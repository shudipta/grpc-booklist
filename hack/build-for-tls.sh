#!/usr/bin/env bash
set -x

# https://stackoverflow.com/a/677212/244009
if [ -x "$(command -v onessl)" ]; then
    export ONESSL=onessl
else
    # ref: https://stackoverflow.com/a/27776822/244009
    case "$(uname -s)" in
        Darwin)
            curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-darwin-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        Linux)
            curl -fsSL -o onessl https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-linux-amd64
            chmod +x onessl
            export ONESSL=./onessl
            ;;

        CYGWIN*|MINGW32*|MSYS*)
            curl -fsSL -o onessl.exe https://github.com/kubepack/onessl/releases/download/0.1.0/onessl-windows-amd64.exe
            chmod +x onessl.exe
            export ONESSL=./onessl.exe
            ;;
        *)
            echo 'other OS'
            ;;
    esac
fi

export REPO_ROOT=$GOPATH/src/github.com/shudipta/grpc-booklist

pushd $REPO_ROOT    
$ONESSL create ca-cert --cert-dir=certs/
$ONESSL create server-cert server --cert-dir=certs/ --domains=localhost
$ONESSL create client-cert client --cert-dir=certs/ --organization=abc.com

go run server/server.go
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