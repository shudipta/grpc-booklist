# grpc-booklist

## Generate gRPC stubs and reverse proxy

```console
protoc -I/usr/local/include -I. \
    -I$GOPATH/src \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --go_out=plugins=grpc:. \
    --grpc-gateway_out=logtostderr=true:. \
    booklist/booklist.proto
```

### Try it out

To compile and run the server, assuming we are in the folder `$GOPATH/src/<path_to_repo_root>`, simply:

```console
go run server/server.go
```

Likewise, to run the client:

```console
go run client/client.go
```

To call RESTful api:

```console
curl --data "{\"name\":\"Sample Book\",\"author\":\"Sample Author\"}" localhost:8080/add
curl localhost:8080/list
```
