# Auction
## Running the application
### Server
- Change the directory to `Hand-in5/Server`
- Write a command in the following format: `go run . -port <port>` where the port can be a integer between 5000-5002.
  For example: `go run . -port 5000`. <br>
IMPORTANT - The initial servers has to be created before the clients makes the first bid.

### Client
- Change the directory to `Hand-in5/Client`
- Write a command in the following format: `go run . -id <ID> -name <name>` where the id is an integer and the name is any string.
  For example: `go run . -id 1 -name John doe`.
  From here you can write two things in the command prompt:

  Bid:      Write any integer to bid that amount. <br>
  Result:   Write `/result` to see server status or winner.