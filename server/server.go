package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/shudipta/grpc-booklist/booklist"
	"github.com/shudipta/grpc-booklist/reverse-proxy"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

// RegisterServicesFunc is a function that registers gRPC services with a given
// server.
type RegisterServicesFunc func(*grpc.Server)

// RegisterServiceHandlerFunc is a function that registers ServiceHandlers with
// a ServeMux.
type RegisterServiceHandlerFunc func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error

// MuxedGRPCServer defines the parameters for running a gRPC Server alongside
// a Gateway server on the same port.
type MuxedGRPCServer struct {
	Addr                string
	TLSConfig           *tls.Config
	ServicesFunc        RegisterServicesFunc
	ServiceHandlerFuncs []RegisterServiceHandlerFunc
}

func ListenAndServe() error {
	srv := MuxedGRPCServer{
		Addr: "127.0.0.1:"+port,
		ServicesFunc: func(gsrv *grpc.Server) {
			pb.RegisterBookListServer(gsrv, &server{})
			reflection.Register(gsrv)
		},
		ServiceHandlerFuncs: []RegisterServiceHandlerFunc{
			pb.RegisterBookListHandler,
		},
	}

	{
		if srv.TLSConfig == nil {
			srv.TLSConfig = &tls.Config{}
		}

		caCert, err := ioutil.ReadFile("certs/ca.crt")
		if err != nil {
			log.Println("error in reading ca cert", err)
			return err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		srv.TLSConfig.ClientCAs = caCertPool
		srv.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert

		cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
		if err != nil {
			log.Println("error in loading server certs", err)
			return err
		}

		srv.TLSConfig.Certificates = []tls.Certificate{cert}
		srv.TLSConfig.NextProtos = []string{"h2"}

		listener, err := tls.Listen("tcp", srv.Addr, srv.TLSConfig)
		if err != nil {
			log.Println("error in listening", err)
			return err
		}

		//fmt.Println(listener.Addr().String())
		//gwHandler, conn, err := NewGateway(listener.Addr().String(), srv.TLSConfig, srv.ServiceHandlerFuncs)
		gwHandler, conn, err := NewGateway(srv.Addr, srv.TLSConfig, srv.ServiceHandlerFuncs)
		if err != nil {
			log.Println("error in creating grpc gateway", err)
			return err
		}
		defer conn.Close()

		gsrv := NewServer(srv.TLSConfig, srv.ServicesFunc)
		defer gsrv.Stop()

		httpHandler := HandlerFunc(gsrv, gwHandler)
		//httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//	log.Println(">>>>>>>> grpc handler")
		//	gsrv.ServeHTTP(w, r)
		//})

		httpsrv := &http.Server{
			Handler: httpHandler,
		}
		httpsrv.Serve(listener)
		//http.ListenAndServe(srv.Addr, httpHandler)
	}

	return nil
}

// NewGateway creates a new http.Handler and grpc.ClientConn with the provided
// gRPC Services registered.
func NewGateway(addr string, tlsConfig *tls.Config, funcs []RegisterServiceHandlerFunc) (http.Handler, *grpc.ClientConn, error) {
	// Configure the right DialOptions the for TLS configuration.
	var dialOpts []grpc.DialOption
	if tlsConfig != nil {
		var gwTLSConfig *tls.Config

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

		gwTLSConfig = &tls.Config{}
		gwTLSConfig.Certificates = []tls.Certificate{certificate}
		gwTLSConfig.RootCAs = certPool

		//gwTLSConfig = tlsConfig.Clone()
		//gwTLSConfig.InsecureSkipVerify = true // Trust the local server.
		//dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(gwTLSConfig)))
		//gwTLSConfig.RootCAs = gwTLSConfig.ClientCAs
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(gwTLSConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Register services.
	srvmux := runtime.NewServeMux()
	for _, fn := range funcs {
		err = fn(ctx, srvmux, conn)
		if err != nil {
			return nil, nil, err
		}
	}


	return srvmux, conn, nil
}

// NewServer allocates a new grpc.Server and handles some some boilerplate
// configuration.
func NewServer(tlsConfig *tls.Config, fn RegisterServicesFunc) *grpc.Server {
	// Default ServerOptions
	grpcOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	}

	if tlsConfig != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	fmt.Println(">>>>>>>>> registering...")
	// Register services with a new grpc.Server.
	gsrv := grpc.NewServer(grpcOpts...)
	fmt.Println(">>>>>>>>> registering1...")
	fn(gsrv)
	fmt.Println(">>>>>>>>> registering2...")

	return gsrv
}

// IsGRPCRequest returns true if the provided request came from a gRPC client.
//
// Its logic is a partial recreation of gRPC's internal checks, see:
// https://github.com/grpc/grpc-go/blob/01de3de/transport/handler_server.go#L61:L69
func IsGRPCRequest(r *http.Request) bool {
	return r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc")
}

// HandlerFunc returns an http.Handler that delegates to grpc.Server on
// incoming gRPC connections otherwise serves with the provided handler.
func HandlerFunc(grpcServer *grpc.Server, otherwise http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsGRPCRequest(r) {
			log.Println(">>>>>>>> grpc handler")
			grpcServer.ServeHTTP(w, r)
		} else {
			log.Println(">>>>>>>> grpc gateway handler")
			otherwise.ServeHTTP(w, r)
		}
	})
}

// ===========================

const (
	port = "10000"
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

func grpcServer()  {
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

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	}

	lis, err := net.Listen("tcp", ":"+port)
	//lis, err := tls.Listen("tcp", ":"+port, tlsConfig)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
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

func main() {
	//grpcServer()
	ListenAndServe()
}
