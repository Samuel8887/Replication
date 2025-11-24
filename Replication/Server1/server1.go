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

	lTime             int64
	currentHighestBid int64

	done        bool
	elapsedTime int64

	lastHeartbeat time.Time
	isLeader      bool

	highestBidder string

	clients []string
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
	s.currentHighestBid = 0
	s.done = false
	s.lastHeartbeat = time.Now()
	s.isLeader = false

	s.grpcServer = grpc.NewServer()
	proto.RegisterReplicationServer(s.grpcServer, s)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			s.clientsMu.Lock()
			if !s.isLeader && time.Since(s.lastHeartbeat) > 10*time.Second {
				s.isLeader = true
				log.Println("Primary down — secondary taking over")
				go func() {
					elapsed := time.Duration(s.elapsedTime) * time.Second
					auctionDuration := 200*time.Second - elapsed
					log.Println("Auction will run for another", auctionDuration)
					time.Sleep(auctionDuration)
					s.clientsMu.Lock()
					s.done = true
					s.clientsMu.Unlock()
					log.Println("Auction is now over")
					log.Printf("The winner is id: %s with bid: %d", s.highestBidder, s.currentHighestBid)

				}()
			}
			s.clientsMu.Unlock()
		}
	}()

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

func (s *Server) Heartbeat(ctx context.Context, req *proto.Heartbeat) (*proto.Empty, error) {
	s.lastHeartbeat = time.Now()
	s.currentHighestBid = req.HighestBid
	s.lTime = req.LogicalTime
	s.done = req.Done
	s.elapsedTime = req.ElapsedTime
	s.highestBidder = req.HighestBidderNow
	s.clients = req.ArrayBids
	return &proto.Empty{}, nil
}

func (s *Server) Bid(ctx context.Context, req *proto.Bid) (*proto.Ack, error) {
	if s.done {
		return &proto.Ack{Success: false, LogicalTime: s.lTime}, nil
	}

	found := false
	for _, n := range s.clients {

		if n == req.ClientId {
			found = true
			break
		}
	}

	if !found {
		s.clients = append(s.clients, req.ClientId)
		log.Printf("First bid from id: %s, added to registry.", req.ClientId)
	}

	if req.MessageBid > s.currentHighestBid {
		s.currentHighestBid = req.MessageBid
		s.highestBidder = req.ClientId
		log.Printf("New highest bid: %d", s.currentHighestBid)
	} else {
		log.Printf("Bid of %d rejected; current highest bid is %d", req.MessageBid, s.currentHighestBid)
		return &proto.Ack{Success: false, LogicalTime: s.lTime}, nil
	}

	return &proto.Ack{Success: true, LogicalTime: s.lTime}, nil
}

func (s *Server) Result(ctx context.Context, req *proto.Result) (*proto.CurrentBid, error) {

	if s.done {
		return &proto.CurrentBid{Money: s.currentHighestBid, LogicalTime: s.lTime, Message: "auction is done"}, nil
	}

	return &proto.CurrentBid{Money: s.currentHighestBid, LogicalTime: s.lTime}, nil
}
