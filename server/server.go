package main

import (
	"auction/auction"
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

type server struct {
	HighestBidderId   int
	HighestBidderName string
	HighestBid        int

	Time float32
	Done bool

	BidMutex sync.Mutex

	auction.UnimplementedAuctionServer
}

func Server() *server {
	return &server{}
}

func main() {
	s := Server()
	s.server()
}

func (s *server) server() {
	server := grpc.NewServer()
	auction.RegisterAuctionServer(server, s)

	listener, error := net.Listen("tcp", ":5000")
	if error != nil {
		log.Fatalf("Failed to listen: %s", error)
	}

	s.timer()

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
					Time:       s.Time,
					HighestBid: int64(s.HighestBid),
				},
			},
		}, nil
	}
}

func (s *server) timer() {
	s.Done = true
}

func (s *server) auction(bid *auction.BidRequest) error {
	if s.Done {
		return fmt.Errorf("auction is done")
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

/*
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
			_, ok := s.Clients[port]

			if ok {
				continue
			}

			connection, error := grpc.Dial(":"+strconv.Itoa(port), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if error != nil {
				continue
			}

			client := auction.NewElectionClient(connection)

			s.Clients[port] = client

			if port > s.Port {
				s.BiggerClients[port] = client
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
	for _, client := range s.BiggerClients {
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

		for _, client := range s.Clients {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			_, _ = client.Coordinator(ctx, coordinatorMessage)
			cancel()
		}
	}
}
*/
