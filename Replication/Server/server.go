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
	currentHighestBid int64

}

func main() {
	server := &Server{}
	server.StartServer()
	for {		
		time.Sleep(5 * time.Second)
		server.lTime++
		log.Printf("Heartbeat sent with logical time %d", server.lTime)
	}


}

func (s *Server) StartServer() {
	lis, err := net.Listen("tcp", ":8000")
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
	if req.MessageBid > s.currentHighestBid {
		s.currentHighestBid = req.MessageBid
		log.Printf("New highest bid: %d at logical time %d", s.currentHighestBid, s.lTime)
	} else {
		log.Printf("Bid of %d rejected; current highest bid is %d at logical time %d", req.MessageBid, s.currentHighestBid, s.lTime)
	}

	return &proto.Ack{Success: true, LogicalTime: s.lTime}, nil
}

func (s* Server) Result(ctx context.Context, req *proto.Result) (*proto.CurrentBid, error) {

	return &proto.CurrentBid{Money: s.currentHighestBid, LogicalTime: s.lTime}, nil
}

