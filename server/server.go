package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	pb "github.com/shudipta/grpc-booklist/booklist"
	"github.com/shudipta/grpc-booklist/reverse-proxy"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":10000"
)

type server struct {
	cnt   int32
	books []*pb.Book
}

func (s *server) Add(ctx context.Context, in *pb.Book) (*pb.AddReply, error) {
	if in.Name == "" || in.Author == "" {
		return &pb.AddReply{Message: "Empty Name or/and Author"}, nil
	}
	s.cnt = s.cnt + 1
	newBook := in
	newBook.Id = s.cnt
	s.books = append(s.books, newBook)
	return &pb.AddReply{Message: fmt.Sprintf("Book '%s' is added successfully.\n", in.Name)}, nil
}

func (s *server) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	return &pb.ListReply{Books: s.books}, nil
}

func main() {
	certificate, err := tls.LoadX509KeyPair(
		"certs/server.crt",
		"certs/server.key",
	)

	certPool := x509.NewCertPool()
	pemCACert, err := ioutil.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("failed to read client ca cert: %s", err)
	}

	ok := certPool.AppendCertsFromPEM(pemCACert)
	if !ok {
		log.Fatal("failed to append client certs")
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	}
	serverOption := grpc.Creds(credentials.NewTLS(tlsConfig))
	s := grpc.NewServer(serverOption)

	// register your server
	pb.RegisterBookListServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	// Run reverse proxy in another goroutine
	go reverse_proxy.Run()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
