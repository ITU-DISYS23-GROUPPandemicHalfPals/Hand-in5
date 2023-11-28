package main

import (
	"auction/auction"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var port = flag.Int("port", 5002, "The port of the node")

type server struct {
	Port            int
	CoordinatorPort int
	Ports           []int

	HighestBidderId   int
	HighestBidderName string
	HighestBid        int

	Time float32
	Done bool

	BidMutex sync.Mutex

	auction.UnimplementedAuctionServer
	auction.UnimplementedElectionServer

	ElectionChannel chan bool

	Servers       map[int]auction.ElectionClient
	BiggerServers map[int]auction.ElectionClient
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

		HighestBid: 50,

		Time: 120,
		Done: false,

		ElectionChannel: make(chan bool, 10),

		Servers:       make(map[int]auction.ElectionClient),
		BiggerServers: make(map[int]auction.ElectionClient),
	}
}

func main() {
	flag.Parse()

	ctx := context.Background()
	s := Server(*port)

	go s.server()
	s.client(ctx)
}

func (s *server) server() {
	server := grpc.NewServer()

	auction.RegisterAuctionServer(server, s)
	auction.RegisterElectionServer(server, s)

	listener, error := net.Listen("tcp", ":"+strconv.Itoa(s.Port))
	if error != nil {
		log.Fatalf("Failed to listen: %s", error)
	}

	go s.timer()

	error = server.Serve(listener)
	if error != nil {
		log.Fatalf("Failed to serve: %s", error)
	}
}

func (s *server) Bid(ctx context.Context, request *auction.BidRequest) (*auction.BidResponse, error) {

	if s.Port != s.CoordinatorPort {

		log.Print("Attempting to connect to coordinator")
		coordinator := s.Servers[s.CoordinatorPort]
		_, error := coordinator.Election(ctx, &auction.ElectionMessage{})

		if error != nil {
			log.Print("No coordinator found: Starting new election")
			s.startElection(ctx)
		}

		if s.Port != s.CoordinatorPort {
			return &auction.BidResponse{}, fmt.Errorf("This is a backup server")
		}

	}

	error := s.auction(request)
	if error != nil {
		return &auction.BidResponse{}, error
	}

	return &auction.BidResponse{}, nil
}

func (s *server) Result(_ context.Context, request *auction.ResultRequest) (*auction.ResultResponse, error) {
	if s.Done {
		return &auction.ResultResponse{
			Event: &auction.ResultResponse_Winner{
				Winner: &auction.ResultResponse_WinnerMessage{
					Name:   s.HighestBidderName,
					Amount: int64(s.HighestBid),
				},
			},
		}, nil
	} else {
		return &auction.ResultResponse{
			Event: &auction.ResultResponse_Status{
				Status: &auction.ResultResponse_StatusMessage{
					Time:       int64(s.Time),
					HighestBid: int64(s.HighestBid),
				},
			},
		}, nil
	}
}

func (s *server) auction(bid *auction.BidRequest) error {

	if s.Done {
		return fmt.Errorf("auction is done")
	}

	if bid.Id == int32(s.HighestBidderId) {
		return fmt.Errorf("you cannot raise your own bid")
	}

	s.BidMutex.Lock()
	defer s.BidMutex.Unlock()
	if int(bid.Amount) > s.HighestBid {
		s.HighestBidderId = int(bid.Id)
		s.HighestBidderName = bid.Name
		s.HighestBid = int(bid.Amount)
	} else {
		return fmt.Errorf("your bid has to be higher than the biggest bid - your bid: %d - highest bid: %d", bid.Amount, s.HighestBid)
	}

	return nil
}

func (s *server) timer() {
	for s.Time > 0 {
		time.Sleep(time.Second)
		s.Time--
	}

	s.Done = true
}

// Election server section ------------------------------------------------------------------------------------------

func (s *server) Election(_ context.Context, request *auction.ElectionMessage) (*auction.Response, error) {
	s.ElectionChannel <- true

	return &auction.Response{}, nil
}

func (s *server) Coordinator(_ context.Context, request *auction.CoordinatorMessage) (*auction.Response, error) {
	s.CoordinatorPort = int(request.Port)
	log.Printf("Made Coordinator request " + strconv.Itoa(s.CoordinatorPort))

	return &auction.Response{}, nil
}

func (s *server) client(ctx context.Context) {
	go s.dialServers()
	go s.broadcastElection(ctx)
	time.Sleep(time.Second)
	s.startElection(ctx)
	for {

	}
}

func (s *server) dialServers() {
	for {
		for _, port := range s.Ports {
			_, ok := s.Servers[port]

			if ok {
				continue
			}

			connection, error := grpc.Dial(":"+strconv.Itoa(port), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if error != nil {
				continue
			}

			server := auction.NewElectionClient(connection)

			s.Servers[port] = server

			if port > s.Port {
				s.BiggerServers[port] = server
			}
		}
		time.Sleep(time.Second)
	}
}

func (s *server) broadcastElection(ctx context.Context) {
	for {
		<-s.ElectionChannel
		s.startElection(ctx)
	}
}

func (s *server) startElection(ctx context.Context) {
	response := false
	for _, server := range s.BiggerServers {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		_, error := server.Election(ctx, &auction.ElectionMessage{})

		if error == nil {
			response = true
		}

		cancel()
	}

	if !response {
		log.Printf("This is now the Coordinator")
		coordinatorMessage := &auction.CoordinatorMessage{
			Port: int32(s.Port),
		}

		for _, server := range s.Servers {
			log.Printf("Running...")
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			_, _ = server.Coordinator(ctx, coordinatorMessage)
			cancel()
		}
		log.Printf(strconv.Itoa(s.CoordinatorPort))
	}
}
