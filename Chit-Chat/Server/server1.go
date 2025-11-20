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
	server := &Server{}
	server.StartServer()
}

func (s *Server) StartServer() {
	lis, err := net.Listen("tcp", ":8001")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s.grpcServer = grpc.NewServer()
	proto.RegisterChit_ChatServer(s.grpcServer, s)

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

func (s *Server) broadcast(msg *proto.ChatMessage, excludeID string) {
	s.clientsMu.Lock()
	clientsCopy := make([]client, len(s.clients))
	copy(clientsCopy, s.clients)
	s.clientsMu.Unlock()
	for _, c := range clientsCopy {
		if c.id == excludeID {
			continue
		}
		go func(c client) {
			if err := c.stream.Send(msg); err != nil {
				log.Printf("Failed to send to %s: %v", c.id, err)
				s.removeClient(c.id)
			}
		}(c)
	}
}


func (s *server) Bid(ctx context.Context, req *proto.Bid) (*proto.Ack, error) {
	return &proto.Ack{Succes: true, LogicalTime: lTime}, nil
}

func (s *server) Result(ctx context.Context, req *proto.Result) (*proto.CurrentBid, error) {
	return &proto.Ack{Money: curretHighestBid, LogicalTime: lTime}, nil
}





func (s *Server) Leave(ctx context.Context, req *proto.LeaveRequest) (*proto.Ack, error) {
	s.clientsMu.Lock()
	var idx int = -1
	for i, c := range s.clients {
		if c.id == req.ClientId {
			idx = i
			break
		}
	}
	if idx != -1 {
		s.clients = append(s.clients[:idx], s.clients[idx+1:]...)
	}
	s.clientsMu.Unlock()

	if req.LogicalTime > s.lTime {
		s.lTime = req.LogicalTime
	}
	s.lTime++

	s.broadcast(&proto.ChatMessage{
		From:        "server",
		Message:     fmt.Sprintf("%s has left the chat", req.ClientId),
		LogicalTime: s.lTime,
	}, req.ClientId)

	return &proto.Ack{Success: true, LogicalTime: s.lTime}, nil
}
