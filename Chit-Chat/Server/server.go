package main

import (
	proto "Chit_Chat/gRPC"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

type client struct {
	id     string
	stream proto.Chit_Chat_JoinServer
}

type Server struct {
	grpcServer *grpc.Server
	proto.UnimplementedChit_ChatServer

	clients   []client
	clientsMu sync.Mutex

	lTime int64
	curretHighestBid int64

}

func main() {
	lTime = 0
	CurrentHighestBid = 0
	
	server := &Server{}
	server.StartServer()


	for {		
		time.Sleep(5 * time.Second)
		s.lTime++
		log.Printf("Heartbeat sent with logical time %d", s.lTime)
	}

}

func (s *Server) StartServer() {
	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s.grpcServer = grpc.NewServer()
	proto.RegisterChit_ChatServer(s.grpcServer, s)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[%s] Server starting on :8000", time.Now().Format(time.RFC3339))
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


func (s* Server) Bid(ctx context.Context, req *proto.Bid) (*proto.Ack, error) {

	return &proto.Ack{Succes: true, LogicalTime: s.lTime}, nil
}

func (s* Server) Result(ctx context.Context, req *proto.Result) (*proto.CurrentBid, error) {

	return &proto.CurrentBid{Money: s.curretHighestBid, LogicalTime: s.lTime}, nil
}

