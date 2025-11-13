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
}

func (s *Server) tickSeen(remote int64) int64 {
	if remote > s.lTime {
		s.lTime = remote
	}
	s.lTime++
	return s.lTime
}

func main() {
	server := &Server{}
	server.StartServer()
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

func (s *Server) removeClient(id string) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	for i, c := range s.clients {
		if c.id == id {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			break
		}
	}
}

func (s *Server) Join(req *proto.JoinRequest, stream proto.Chit_Chat_JoinServer) error {
	s.clientsMu.Lock()
	s.clients = append(s.clients, client{id: req.ClientId, stream: stream})
	s.clientsMu.Unlock()

	if req.LogicalTime > s.lTime {
		s.lTime = req.LogicalTime
	}
	s.lTime++

	s.broadcast(&proto.ChatMessage{
		From:        "server",
		Message:     fmt.Sprintf("%s has joined the chat!", req.ClientId),
		LogicalTime: s.lTime,
	}, req.ClientId)

	select {}
}

func (s *Server) Publish(ctx context.Context, req *proto.PublishRequest) (*proto.Ack, error) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if req.LogicalTime > s.lTime {
		s.lTime = req.LogicalTime
	}
	s.lTime++

	msg := &proto.ChatMessage{
		From:        req.ClientId,
		Message:     req.Content,
		LogicalTime: s.lTime,
	}

	for i := 0; i < len(s.clients); i++ {
		if s.clients[i].id == req.ClientId {
			continue
		}
		if err := s.clients[i].stream.Send(msg); err != nil {
			log.Printf("failed to send to %s: %v", s.clients[i].id, err)
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			i--
		}
	}

	return &proto.Ack{Success: true, LogicalTime: s.lTime}, nil
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
