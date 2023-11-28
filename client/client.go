package main

import (
	"auction/auction"
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var id = flag.Int("id", 1, "The id of the client")
var name = flag.String("name", "John Doe", "The name of the client")

type client struct {
	Id   int
	Name string

	Clients []auction.AuctionClient
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

	for i := 5000; i <= 5002; i++ {
		connection, error := grpc.Dial(":"+strconv.Itoa(i), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if error != nil {
			log.Fatalf("Connecting to server failed: %s", error)
		}

		c.Clients = append(c.Clients, auction.NewAuctionClient(connection))
	}

	c.run(ctx)
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
	var responses map[string]int = make(map[string]int)
	var mostOccuringResponse *auction.ResultResponse
	var mostOcurrences int
	var secondMostOcurrences int

	for _, client := range c.Clients {
		response, error := client.Result(ctx, &auction.ResultRequest{})
		if error != nil {
			continue
		}

		i, ok := responses[response.String()]
		if ok {
			responses[response.String()] = i + 1
			if i+1 > mostOcurrences {
				mostOccuringResponse = response
				mostOcurrences = i + 1
			} else if i+1 > secondMostOcurrences {
				secondMostOcurrences = i + 1
			}
		} else {
			responses[response.String()] = 1
			if 1 > mostOcurrences {
				mostOccuringResponse = response
				mostOcurrences = 1
			} else if 1 > secondMostOcurrences {
				secondMostOcurrences = 1
			}
		}
	}

	if mostOccuringResponse == nil {
		log.Printf("No response from the server")
	} else if mostOcurrences > secondMostOcurrences {
		switch event := mostOccuringResponse.Event.(type) {
		case *auction.ResultResponse_Status:
			log.Printf("The highest bid is %d. There are %d seconds left of the auction.", event.Status.HighestBid, event.Status.Time)
		case *auction.ResultResponse_Winner:
			log.Printf("The auction is over. The winning bid is %d by %s", event.Winner.Amount, event.Winner.Name)
		}
	} else {
		log.Printf("Mismatched response from the server")
	}
}

func (c *client) bid(ctx context.Context, bidAmount int) {
	var errors []error
	for _, client := range c.Clients {
		_, error := client.Bid(ctx, &auction.BidRequest{
			Id:     int32(c.Id),
			Name:   c.Name,
			Amount: int64(bidAmount),
		})

		if error != nil {
			errors = append(errors, error)
		}
	}

	if len(errors) == 3 {
		for _, error := range errors {
			log.Print(error)
		}
	} else {
		log.Printf("Successfully placed bid")
	}
}
