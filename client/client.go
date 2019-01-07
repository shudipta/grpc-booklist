package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"time"

	pb "github.com/shudipta/grpc-booklist/booklist"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	address = "localhost:10000"
	// address = "192.168.99.100:30010"
)

func add(c pb.BookListClient, book *pb.Book) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, err := c.Add(ctx, book)
	if err != nil {
		log.Fatalf("%v.Add(_) = _, %v", c, err)
	}
	log.Println(r)
}

func list(c pb.BookListClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.List(ctx, &pb.ListRequest{})
	if err != nil {
		log.Fatalf("%v.List(_) = _, %v", c, err)
	}
	log.Println(r)
}

func main() {
	certificate, err := tls.LoadX509KeyPair(
		"certs/client.crt",
		"certs/client.key",
	)

	certPool := x509.NewCertPool()
	pemCACert, err := ioutil.ReadFile("certs/ca.crt")
	if err != nil {
		log.Fatalf("failed to read ca cert: %s", err)
	}

	ok := certPool.AppendCertsFromPEM(pemCACert)
	if !ok {
		log.Fatal("failed to append certs")
	}

	transportCreds := credentials.NewTLS(&tls.Config{
		// ServerName:   "example.com",
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	dialOption := grpc.WithTransportCredentials(transportCreds)

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, dialOption)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewBookListClient(conn)

	// Contact the server and print out its response.
	add(c, &pb.Book{Name: "aaa", Author: "AAA"})
	add(c, &pb.Book{Name: "bbb", Author: "BBB"})
	add(c, &pb.Book{Name: "ccc", Author: "CCC"})

	list(c)
}
