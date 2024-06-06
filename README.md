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

## Implementation details
- the server is written in `go`;
- client interactivity is achieved through `htmx` + websockets;

## TODO
- [ ] associate websocket connection uids with usernames;
- [ ] update auction list whenever an auction expires;
