package main

import (
	"auction/auction"
	"context"
	"flag"
	"log"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var port = flag.Int("port", 5000, "The port of the server")

type server struct {
	Port            int
	CoordinatorPort int
	Ports           []int

	ElectionChannel chan bool
	Clients         map[int]auction.ElectionClient
	BiggerClients   map[int]auction.ElectionClient

	auction.UnimplementedElectionServer
}

func Server(port int) *server {
	var ports []int

	for i := 5000; i <= 5002; i++ {
		ports = append(ports, i)
	}

	return &server{
		Port:            port,
		CoordinatorPort: 0,
		Ports:           ports,

		ElectionChannel: make(chan bool, 10),
		Clients:         make(map[int]auction.ElectionClient),
		BiggerClients:   make(map[int]auction.ElectionClient),
	}
}

func main() {
	flag.Parse()

	n := Server(*port)

	ctx := context.Background()

	go n.server(ctx)
	n.client(ctx)
}

func (n *server) server(ctx context.Context) {
	s := grpc.NewServer()
	auction.RegisterElectionServer(s, n)

	listener, error := net.Listen("tcp", ":"+strconv.Itoa(n.Port))
	if error != nil {
		log.Fatalf("Failed to listen: %s", error)
	}

	error = s.Serve(listener)
	if error != nil {
		log.Fatalf("Failed to serve: %s", error)
	}
}

func (n *server) Election(_ context.Context, request *auction.ElectionMessage) (*auction.Response, error) {
	n.ElectionChannel <- true

	return &auction.Response{}, nil
}

func (n *server) Coordinator(_ context.Context, request *auction.CoordinatorMessage) (*auction.Response, error) {
	n.CoordinatorPort = int(request.Port)

	return &auction.Response{}, nil
}

func (n *server) client(ctx context.Context) {
	go n.dialServers()
	n.broadcastElection(ctx)
}

func (n *server) dialServers() {
	for {
		time.Sleep(time.Second)

		for _, port := range n.Ports {
			_, ok := n.Clients[port]

			if ok {
				continue
			}

			connection, error := grpc.Dial(":"+strconv.Itoa(port), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if error != nil {
				continue
			}

			client := auction.NewElectionClient(connection)

			n.Clients[port] = client

			if port > n.Port {
				n.BiggerClients[port] = client
			}
		}
	}
}

func (n *server) broadcastElection(ctx context.Context) {
	for {
		<-n.ElectionChannel
		n.startElection(ctx)
	}
}

func (n *server) startElection(ctx context.Context) {
	response := false
	for _, client := range n.BiggerClients {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		_, error := client.Election(ctx, &auction.ElectionMessage{})

		if error == nil {
			response = true
		}

		cancel()
	}

	if !response {
		coordinatorMessage := &auction.CoordinatorMessage{
			Port: int32(n.Port),
		}

		for _, client := range n.Clients {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			_, _ = client.Coordinator(ctx, coordinatorMessage)
			cancel()
		}
	}
}
