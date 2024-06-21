# Auction

## Description
This app allows emulating an auction functionality. An auction is an example of an application that
supports more than one user participating in the same activity at the same time. All the participants
are updated about the other users' activity (in this case -- new highest bid placed) instantly.

## Functionality
- admin capabilities (creating new auctions with set expiration);
- every user is able to see a list of all currently running auctions;
- every user can join a particular auction, observe its state (time till expiration, current highest
bid), place new bids (invalid bids are rejected);

## Usage
1. clone the repo to your local directory with `git clone git@github.com:dogz1lla/auction.git` [^1]
2. `cd` into the newly cloned directory and run `go run cmd/auction/main.go` [^2]
3. in your browser navigate to `localhost:3000/login` and enter `admin` to have admin privilegies or
any other username for ordinary user privilegies

[^1]: make sure you have git through ssh access configured
[^2]: make sure you have `go` installed and updated

## Implementation details
- the server is written in `go`;
- client interactivity is achieved through `htmx` + websockets;

## TODO
- [x] ~~associate websocket connection uids with usernames;~~
- [x] ~~update auction list whenever an auction expires;~~
