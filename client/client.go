package main

import (
	"auction/auction"
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var id = flag.Int("id", 1, "The id of the client")
var name = flag.String("name", "John Doe", "The name of the client")

type client struct {
	Id   int
	Name string

	auction.AuctionClient
}

func Client(id int, name string) *client {
	return &client{
		Id:   id,
		Name: name,
	}
}

func main() {
	flag.Parse()

	c := Client(*id, *name)
	c.client()
}

func (c *client) client() {
	ctx := context.Background()

	c.connectToServer(ctx)

	for {

	}
}

func (c *client) run(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if scanner.Scan() {
			text := scanner.Text()

			if text == "/result" {
				c.result(ctx)
				continue
			}

			bidAmount, error := strconv.Atoi(text)
			if error != nil {
				log.Print("not a valid bid")
				continue
			}

			c.bid(ctx, bidAmount)
		}
	}
}

func (c *client) result(ctx context.Context) {
	response, error := c.AuctionClient.Result(ctx, &auction.ResultRequest{})
	if error != nil {
		log.Print("Connection lost - Attempting to reconnect")
		c.connectToServer(ctx)
		return
	}

	switch event := response.Event.(type) {
	case *auction.ResultResponse_Status:
		log.Printf("The highest bid is %d. There are %d seconds left of the auction.", event.Status.HighestBid, event.Status.Time)
	case *auction.ResultResponse_Winner:
		log.Printf("The auction is over. The winning bid is %d by %s", event.Winner.Amount, event.Winner.Name)
	}
}

func (c *client) bid(ctx context.Context, bidAmount int) {
	_, error := c.AuctionClient.Bid(ctx, &auction.BidRequest{
		Id:     int32(c.Id),
		Name:   c.Name,
		Amount: int64(bidAmount),
	})
	if error != nil {
		log.Print("Connection lost - Attempting to reconnect")
		c.connectToServer(ctx)
	} else {
		log.Print("Successfully placed bid")
	}
}

func (c *client) connectToServer(ctx context.Context) {

	var serverPort = 5002

	for serverPort >= 5000 {
		connection, err := grpc.Dial(":"+strconv.Itoa(serverPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithTimeout(3*time.Second),
		)

		if err == nil {
			c.AuctionClient = auction.NewAuctionClient(connection)
			log.Printf("Connection established to port " + strconv.Itoa(serverPort))

			c.run(ctx)
		}
		log.Printf("Couldn't find port " + strconv.Itoa(serverPort))
		serverPort--
	}

	log.Fatalf("Failed to connect to any server")

}
