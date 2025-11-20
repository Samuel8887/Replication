package main

import (
	proto "Replication/gRPC"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}

	client := proto.NewReplicationClient(conn)
	reader := bufio.NewReader(os.Stdin)
	logicalTime := 0

	
	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("input went wrong %v", err)
		}
		input = strings.TrimSpace(input)
		parts := strings.SplitN(input, " ", 2)
		command := parts[0]
		var bud int64
		if len(parts) > 1 {
			tmp, err := strconv.ParseInt(parts[1], 10, 64)
        if err != nil {
            fmt.Println("Invalid bid amount:", parts[1])
            continue
        }
        bud = tmp
		}
		if command == "bid" {
			logicalTime++
			response, err := client.Bid(context.Background(), &proto.Bid{
				ClientId: "P1", LogicalTime: int64(logicalTime), MessageBid: int64(bud),
			})
			if err != nil {
				//log.Println("Error placing bid: %v", err)
				conn1, err := grpc.NewClient("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
            		log.Fatalf("Failed to connect to backup server: %v", err)
        		}
				client1 := proto.NewReplicationClient(conn1)
				response, err = client1.Bid(context.Background(), &proto.Bid{
				ClientId: "P1", LogicalTime: int64(logicalTime), MessageBid: int64(bud),
				})
				if err != nil {
            		log.Fatalf("Backup server also failed: %v", err)
        		}
			}
			fmt.Printf("Bid Response: %t (Logical Time: %d)\n", response.Success, response.LogicalTime)
		}

		if command == "result" {
			logicalTime++
			response, err := client.Result(context.Background(), &proto.Result{
				ClientId: "P1", LogicalTime: int64(logicalTime),
			})
			if err != nil {
				//log.Println("Error placing bid: %v", err)
				conn1, err := grpc.NewClient("localhost:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
            		log.Fatalf("Failed to connect to backup server: %v", err)
        		}
				client1 := proto.NewReplicationClient(conn1)
				response, err = client1.Result(context.Background(), &proto.Result{
				ClientId: "P1", LogicalTime: int64(logicalTime),
				})
				if err != nil {
            		log.Fatalf("Backup server also failed: %v", err)
        		}

			}
			fmt.Printf("Current Highest Bid: %d (Logical Time: %d)\n", response.Money, response.LogicalTime)
			if response.Message != "" {
				fmt.Printf("Message from server: %s\n", response.Message)
			}

		}
	}
}
