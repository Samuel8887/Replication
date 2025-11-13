package main

import (
	proto "Chit_Chat/gRPC"
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}

	client := proto.NewChit_ChatClient(conn)
	reader := bufio.NewReader(os.Stdin)
	logicalTime := 0

	joined := false
	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("input went wrong %v", err)
		}
		input = strings.TrimSpace(input)
		parts := strings.SplitN(input, " ", 2)
		command := parts[0]
		message := ""
		if len(parts) > 1 {
			message = parts[1]
		}
		if command == "joinP1" {
			if !joined {
				stream, err := client.Join(context.Background(), &proto.JoinRequest{
					ClientId: "P1", LogicalTime: int64(logicalTime + 1),
				})
				if err != nil {
					log.Fatalf("could not join: %v", err)
				}

				joined = true

				go func() {
					for {
						msg, err := stream.Recv()
						if err == io.EOF {
							log.Println("Stream closed")
							break
						}
						if err != nil {
							log.Fatalf("Server connection lost")
						}
						log.Printf("received: %v", msg)
					}
				}()
			} else {
				log.Printf("Already joined!")
			}
		}

		if command == "publishP1" {
			if !joined {
				log.Println("You are not in the chat!")
				continue
			}
			_, _ = client.Publish(context.Background(), &proto.PublishRequest{
				ClientId: "P1", LogicalTime: int64(logicalTime + 1), Content: message,
			})
			//log.Printf("Publishing: %v", publishRequest)
		}
		if command == "leaveP1" {
			if !joined {
				log.Println("You are not in the chat!")
				continue
			}
			_, _ = client.Leave(context.Background(), &proto.LeaveRequest{
				ClientId: "P1", LogicalTime: int64(logicalTime + 1),
			})
			//log.Printf("Leaving: %v", leaveRequest)
			joined = false
		}
	}
}
