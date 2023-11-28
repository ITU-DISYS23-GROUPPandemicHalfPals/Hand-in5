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
)

var port = flag.Int("port", 5000, "The id of the client")

type server struct {
	Port int

	HighestBidderId   int
	HighestBidderName string
	HighestBid        int

	Time int

	Started  bool
	Finished bool

	BidMutex sync.Mutex

	auction.UnimplementedAuctionServer
}

func Server(port int) *server {
	return &server{
		Port: port,

		HighestBid: 50,
		Time:       120,

		Started:  false,
		Finished: false,
	}
}

func main() {
	flag.Parse()

	s := Server(*port)
	s.server()
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

	if !s.Started {
		s.Started = true
	}

	return &auction.BidResponse{}, nil
}

func (s *server) Result(_ context.Context, request *auction.ResultRequest) (*auction.ResultResponse, error) {
	if !s.Started {
		return &auction.ResultResponse{
			Event: &auction.ResultResponse_Status{
				Status: &auction.ResultResponse_StatusMessage{
					Time:       int64(s.Time),
					HighestBid: int64(s.HighestBid),
				},
			},
		}, nil
	} else if s.Finished {
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
	if s.Finished {
		return fmt.Errorf("auction is done")
	}

	if bid.Id == int32(s.HighestBidderId) {
		return fmt.Errorf("you can not raise your own bid")
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
	for !s.Started {

	}

	for s.Time > 0 {
		time.Sleep(time.Second)
		s.Time--
	}

	s.Finished = true
}
