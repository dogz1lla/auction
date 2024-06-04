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

Need to add logic that can notify users about changes in the auction room state and whether there
are new auctions rooms or if any of the existing ones are expired;
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
	ClosesAt time.Time
}

func NewAuctionRoom() *AuctionRoom {
	id := uuid.New()
	return &AuctionRoom{
		Id:            id.String(),
		CurrentBidder: "none",
		CurrentBid:    0.0,
		// ClosesAt:      0,
	}
}

func NewMockAuctionRoom() *AuctionRoom {
	// TODO delete
	return &AuctionRoom{
		Id:            "lul",
		CurrentBidder: "none",
		CurrentBid:    0.0,
		ClosesAt:      time.Now().UTC().Add(10 * time.Minute),
	}
}

func (ar *AuctionRoom) ProcessBid(userName string, msg *Message) error {
	if time.Now().UTC().After(ar.ClosesAt) {
		log.Printf("Room %s has expired! No new bids allowed\n", ar.Id)
		return nil
	}
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

func (ar *AuctionRoom) RenderRoomListEntry() []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	// DONE: fix the div id naming
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "home-auction-entry", ar)
	if err != nil {
		log.Fatalf("Template parsing error: %s", err)
	}

	return renderedMsg.Bytes()
}

func (ar *AuctionRoom) RenderNewRoomEntry() []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	// DONE: fix the div id naming
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "appendable-auction-entry", ar)
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
	Rooms map[string]*AuctionRoom

	register   chan *AuctionRoom
	unregister chan *AuctionRoom
}

func NewRoomManager() *RoomManager {
	// TODO fix
	testRooms := make(map[string]*AuctionRoom)
	testRooms["lul"] = NewMockAuctionRoom()
	return &RoomManager{
		Rooms:      testRooms,
		register:   make(chan *AuctionRoom),
		unregister: make(chan *AuctionRoom),
	}
}

func (rm *RoomManager) Run() {
	for {
		select {
		case auctionRoom := <-rm.register:
			// NOTE maps in go are not concurrent so use the lock (mentioned at 24:15 in the video)
			rm.Rooms[auctionRoom.Id] = auctionRoom
			log.Printf("room registered: %s\n", auctionRoom.Id)
		case auctionRoom := <-rm.unregister:
			if _, ok := rm.Rooms[auctionRoom.Id]; ok {
				delete(rm.Rooms, auctionRoom.Id)
				log.Printf("room unregistered: %s\n", auctionRoom.Id)
			}
		}
	}
}

func (rm *RoomManager) GetRoomById(roomId string) (*AuctionRoom, error) {
	if room, ok := rm.Rooms[roomId]; ok {
		return room, nil
	}
	return nil, errors.New(fmt.Sprintf("Auction room %s not found", roomId))
}

func (rm *RoomManager) CreateAuction(closesAt time.Time) *AuctionRoom {
	newAuctionRoom := NewAuctionRoom()
	newAuctionRoom.ClosesAt = closesAt
	rm.Rooms[newAuctionRoom.Id] = newAuctionRoom
	return newAuctionRoom
}

// have to put it here instead of templating because of circular imports
type AuctionPage struct {
	User       *users.User
	Room       *AuctionRoom
	Expiration int64
}

func NewAuctionPage(ar *AuctionRoom) *AuctionPage {
	return &AuctionPage{
		User:       users.NewUser("MOCK USER"),
		Room:       ar,
		Expiration: GetMillisTill(ar.ClosesAt),
	}
}
func NewMockAuctionPage(rm *RoomManager) *AuctionPage {
	mockRoom := rm.Rooms["lul"]
	return &AuctionPage{
		User:       users.NewUser("MOCK USER"),
		Room:       mockRoom,
		Expiration: GetMillisTill(mockRoom.ClosesAt),
	}
}

// NOTE: again, need to define this here instead of in templating due to the circular imports
type HomePage struct {
	IsAdmin bool
	Rm      *RoomManager
}

func NewHomePage(isAdmin bool, rm *RoomManager) HomePage {
	return HomePage{IsAdmin: isAdmin, Rm: rm}
}
