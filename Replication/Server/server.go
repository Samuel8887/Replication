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

	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc"
)



type Server struct {
	grpcServer *grpc.Server
	proto.UnimplementedReplicationServer

	clientsMu sync.Mutex

	lTime int64
	currentHighestBid int64

	done bool
	startTime time.Time
	sendHeartbeatsToSecondary func()
	elapsedTime int64


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

	conn, err := grpc.NewClient("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("failed to connect to secondary server: %v", err)
    }
    secondaryClient := proto.NewReplicationClient(conn)

	s.lTime = 0
	s.currentHighestBid = 0
	s.done = false
	s.startTime = time.Now()

	s.grpcServer = grpc.NewServer()
	proto.RegisterReplicationServer(s.grpcServer, s)

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		
		for {
			<-ticker.C
			s.clientsMu.Lock()
			s.elapsedTime = int64(time.Since(s.startTime).Seconds())
			highest := s.currentHighestBid
			s.clientsMu.Unlock()
			_, err := secondaryClient.Heartbeat(context.Background(), &proto.Heartbeat{
				Succes:   true,
				LogicalTime: s.lTime,
				HighestBid: highest,
				ElapsedTime: s.elapsedTime,
				Done: 	s.done,
			})
			if err != nil {
				log.Println("Failed to send heartbeat to secondary:", err)
			}
		}
	}()

	go func() {
        auctionDuration := 200 * time.Second
        time.Sleep(auctionDuration)
        s.clientsMu.Lock()
        s.done = true
        s.clientsMu.Unlock()
        log.Println("Auction is now over")
    }()

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
	if s.done {
		return &proto.Ack{Success: false, LogicalTime: s.lTime}, nil
	}

	if req.MessageBid > s.currentHighestBid {
		s.currentHighestBid = req.MessageBid
		log.Printf("New highest bid: %d at logical time %d", s.currentHighestBid, s.lTime)
	} else {
		log.Printf("Bid of %d rejected; current highest bid is %d at logical time %d", req.MessageBid, s.currentHighestBid, s.lTime)
		return &proto.Ack{Success: false, LogicalTime: s.lTime}, nil
	}

	return &proto.Ack{Success: true, LogicalTime: s.lTime}, nil
}

func (s* Server) Result(ctx context.Context, req *proto.Result) (*proto.CurrentBid, error) {

	if s.done {
		return &proto.CurrentBid{Money: s.currentHighestBid, LogicalTime: s.lTime, Message: "auction is done"}, nil
	}

	return &proto.CurrentBid{Money: s.currentHighestBid, LogicalTime: s.lTime}, nil
}

