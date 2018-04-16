package main

import (
	"log"
	"time"

	pb "github.com/shudipta/grpc-booklist/booklist"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	address = "localhost:10000"
)

func add(c pb.BookListClient, book *pb.Book) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
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
