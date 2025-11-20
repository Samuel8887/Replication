package main

import (
	proto "Replication/gRPC"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
)


type Server struct {
	grpcServer *grpc.Server
	proto.UnimplementedReplicationServer

	clientsMu sync.Mutex

	lTime int64
	curretHighestBid int64
}



func main() {
	server := &Server{}
	server.StartServer()
}

func (s *Server) StartServer() {
	lis, err := net.Listen("tcp", ":8001")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s.lTime = 0
	s.curretHighestBid = 0


	s.grpcServer = grpc.NewServer()
	proto.RegisterReplicationServer(s.grpcServer, s)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[%s] Server starting on :8001", time.Now().Format(time.RFC3339))
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	<-stop
	s.StopServer()
}

func (s *Server) StopServer() {
	s.grpcServer.Stop()
	log.Println("Server stopped")
}


func (s *Server) Bid(ctx context.Context, req *proto.Bid) (*proto.Ack, error) {
	return &proto.Ack{Success: true, LogicalTime: s.lTime}, nil
}

func (s *Server) Result(ctx context.Context, req *proto.Result) (*proto.CurrentBid, error) {
	return &proto.CurrentBid{Money: s.curretHighestBid, LogicalTime: s.lTime}, nil
}






