# Auction
## Running the application
**IMPORTANT** - The initial servers has to be created before the clients makes the first bid.

### Server
- Change the directory to `Hand-in5/Server`
- Write a command in the following format: `go run . -port <port>`. <br>
The port has to be an unique integer between 5000-5002. <br>
For example: `go run . -port 5000`. <br>

### Client
- Change the directory to `Hand-in5/Client`.
- Write a command in the following format: `go run . -id <ID> -name <name>` where the id is an unique integer and the name is any string. <br>
For example: `go run . -id 1 -name John doe`.
- You can now write one of the following commands: <br>
  - **Bid**:      Write any integer to bid that amount.  
  - **Result**:   Write `/result` to see server status or winner.
