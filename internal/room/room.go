/*
Room should have an expiry timestamp;
room can be created and that should be communicated to the hub of clients?
when the room expires the list of rooms view should be notified (? js should update the status on
the client)
register chan *AuctionRoom

Auction room must be able to exist whether there is even a single client in it or not; so does the
room manager
TODO
- [ ] ;
*/
package room

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/dogz1lla/auction/internal/templating"
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
	// this will help render the countdown both in user's list and in the auction view
	closesAt int64
}

func NewAuctionRoom() *AuctionRoom {
	id := uuid.New()
	return &AuctionRoom{
		id:            id.String(),
		CurrentBidder: "none",
		CurrentBid:    0.0,
		closesAt:      0,
	}
}

func onBid() error {
	return nil
}

func (ar *AuctionRoom) ProcessBid(userName string, msg *Message) error {
	bid := msg.Bid
	if bid > ar.CurrentBid {
		ar.CurrentBid = bid
		ar.CurrentBidder = msg.WsClient.id
		log.Printf("New bid! %s bid %f\n", msg.WsClient.id, bid)
	} else {
		log.Printf("Bid rejected! %s bid %f\n", msg.WsClient.id, bid)
	}
	return nil
}

// Render the html template for the auction's state as slice of bytes
func (ar *AuctionRoom) RenderState() []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "auction-state", ar)
	if err != nil {
		log.Fatalf("Template parsing error: %s", err)
	}

	return renderedMsg.Bytes()
}

// To have more than one room we need a room manager.
// The room manager should maintain a collection of currently active rooms;
// it should also be able to register new rooms and remove rooms;
// NOTE there is only one room per ws connection; can i abuse that? :thinking:
// the room id could be extracted from the list of the auctions on the user's home page;
// i can send the room id to the /ws endpoint and thus make it a part of the Client struct
type RoomManager struct {
	rooms map[string]*AuctionRoom

	register   chan *AuctionRoom
	unregister chan *AuctionRoom
}

func NewRoomManager() *RoomManager {
	// TODO fix
	testRooms := make(map[string]*AuctionRoom)
	testRooms["lul"] = NewAuctionRoom()
	return &RoomManager{
		rooms:      testRooms,
		register:   make(chan *AuctionRoom),
		unregister: make(chan *AuctionRoom),
	}
}

func (rm *RoomManager) Run() {
	for {
		select {
		case auctionRoom := <-rm.register:
			// NOTE maps in go are not concurrent so use the lock (mentioned at 24:15 in the video)
			rm.rooms[auctionRoom.id] = auctionRoom
			log.Printf("room registered: %s\n", auctionRoom.id)
		case auctionRoom := <-rm.unregister:
			if _, ok := rm.rooms[auctionRoom.id]; ok {
				delete(rm.rooms, auctionRoom.id)
				log.Printf("room unregistered: %s\n", auctionRoom.id)
			}
		}
	}
}

func (rm *RoomManager) getRoomById(roomId string) (*AuctionRoom, error) {
	if room, ok := rm.rooms[roomId]; ok {
		return room, nil
	}
	return nil, errors.New(fmt.Sprintf("Auction room %s not found", roomId))
}
