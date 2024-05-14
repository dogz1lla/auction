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
	"time"

	"github.com/dogz1lla/auction/internal/templating"
	"github.com/dogz1lla/auction/internal/users"
	"github.com/google/uuid"
)

// type BidMsg struct {
// 	BidderId string  `json:"bidderId"`
// 	Bid      float64 `json:"bid,string"`
// }

type AuctionRoom struct {
	Id            string
	CurrentBidder string
	CurrentBid    float64
	// this will help render the countdown both in user's list and in the auction view
	closesAt time.Time
}

func NewAuctionRoom() *AuctionRoom {
	id := uuid.New()
	return &AuctionRoom{
		Id:            id.String(),
		CurrentBidder: "none",
		CurrentBid:    0.0,
		// closesAt:      0,
	}
}

func NewMockAuctionRoom() *AuctionRoom {
	// TODO delete
	return &AuctionRoom{
		Id:            "lul",
		CurrentBidder: "none",
		CurrentBid:    0.0,
		closesAt:      time.Now().UTC().Add(10 * time.Minute),
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
	testRooms["lul"] = NewMockAuctionRoom()
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
			rm.rooms[auctionRoom.Id] = auctionRoom
			log.Printf("room registered: %s\n", auctionRoom.Id)
		case auctionRoom := <-rm.unregister:
			if _, ok := rm.rooms[auctionRoom.Id]; ok {
				delete(rm.rooms, auctionRoom.Id)
				log.Printf("room unregistered: %s\n", auctionRoom.Id)
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

// have to put it here instead of templating because of circular imports
type AuctionPage struct {
	User       *users.User
	Room       *AuctionRoom
	Expiration int64
}

func NewMockAuctionPage(rm *RoomManager) *AuctionPage {
	mockRoom := rm.rooms["lul"]
	return &AuctionPage{
		User:       users.NewUser("MOCK USER"),
		Room:       mockRoom,
		Expiration: GetMillisTill(mockRoom.closesAt),
	}
}
