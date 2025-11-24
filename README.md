# Replication

1. Start by running server.go
2. Then run server1.go (the backup) in a seperate terminal
3. Afterwards run ClientP1.go, ClientP2.go, and ClientP3.go in seperate terminals to start the clients.
4. To bid, in a client terminal, type "bid [ammount]" where ammount is the money to bid. Bidding under will result in a failure. 
5. To get the current highest bid, type "result"
6. If server.go crashes/terminates, server1.go will take over"
7. After 200 seconds, the highest bid and bidder will be anounced in the server terminal
8. To terminate the servers or clients, press: CTRL+C in their specific terminals
