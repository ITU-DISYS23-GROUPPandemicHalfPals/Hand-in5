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

	ElectionChannel chan bool

	Servers       map[int]auction.ElectionClient
	BiggerServers map[int]auction.ElectionClient
}

func Server(port int) *server {
	return &server{
		Port:       port,
		HighestBid: 50,

		Time: 120,
		Done: false,
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

func (s *server) Bid(_ context.Context, request *auction.BidRequest) (*auction.BidResponse, error) {
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

	if s.Port != s.CoordinatorPort {
		return fmt.Errorf("This is a backup server")
	}

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

	return &auction.Response{}, nil
}

func (s *server) client(ctx context.Context) {
	go s.dialServers()
	s.broadcastElection(ctx)
}

func (s *server) dialServers() {
	for {
		time.Sleep(time.Second)

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
	for _, client := range s.BiggerServers {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		_, error := client.Election(ctx, &auction.ElectionMessage{})

		if error == nil {
			response = true
		}

		cancel()
	}

	if !response {
		coordinatorMessage := &auction.CoordinatorMessage{
			Port: int32(s.Port),
		}

		for _, client := range s.Servers {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			_, _ = client.Coordinator(ctx, coordinatorMessage)
			cancel()
		}
	}
}
