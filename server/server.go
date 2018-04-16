package main

import (
	"fmt"
	"log"
	"net"

	pb "github.com/shudipta/grpc-booklist/booklist"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterBookListServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
