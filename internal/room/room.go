/*
TODO
- [x] map from uid to a channel;
- [x] "connectRoom" method that uses user name and creates a new channel;
- [ ] on connect: listen on a bid message; send nothing back but have a callback for that;
- [ ] have a on disconnect method;
*/
package room

import (
	"log"

	"github.com/google/uuid"
)

// type BidMsg struct {
// 	BidderId string  `json:"bidderId"`
// 	Bid      float64 `json:"bid,string"`
// }

type AuctionRoom struct {
	id            string
	CurrentBidder string
	CurrentBid    float64
}

// type AuctionHouse struct {
// 	Rooms map[string]*AuctionRoom
// }
//
// func NewAuctionHouse() *AuctionHouse {
// 	return &AuctionHouse{
// 		Rooms: make(map[string]*AuctionRoom),
// 	}
// }

func NewAuctionRoom() *AuctionRoom {
	id := uuid.New()
	return &AuctionRoom{
		id:            id.String(),
		CurrentBidder: "none",
		CurrentBid:    0.0,
	}
}

func onBid() error {
	return nil
}

func (ar *AuctionRoom) ProcessBid(userName string, msg *Message) error {
	bid := msg.Bid
	if bid > ar.CurrentBid {
		ar.CurrentBid = bid
		ar.CurrentBidder = msg.ClientID
		log.Printf("New bid! %s bid %f\n", msg.ClientID, bid)
	} else {
		log.Printf("Bid rejected! %s bid %f\n", msg.ClientID, bid)
	}
	return nil
}

// To have more than one room we need a room manager.
// The room manager should maintain a collection of currently active rooms;
// it should also be able to register new rooms and remove rooms;
// NOTE there is only one room per ws connection; can i abuse that? :thinking:
// the room id could be extracted from the list of the auctions on the user's home page;
// i can send the room id to the /ws endpoint and thus make it a part of the Client struct
type RoomManager struct {
	rooms map[string]*AuctionRoom

	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}
