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

func (ar *AuctionRoom) ProcessBid(userName string, msg *Message) error {
	if time.Now().UTC().After(ar.ClosesAt) {
		log.Printf("Room %s has expired! No new bids allowed!\n", ar.Id)
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

// Below is code related to auction list found on the home page of a user
type RoomListEntry struct {
	// TODO: compare with the AuctionPage, remove redundancy
	// TODO: in the auction_list_body element need to make sure that the collection is of entries
	Room      *AuctionRoom
	ExpiresIn int64
}

func NewRoomListEntry(room *AuctionRoom) *RoomListEntry {
	return &RoomListEntry{
		Room:      room,
		ExpiresIn: GetMillisTill(room.ClosesAt),
	}
}

func (ar *AuctionRoom) RenderRoomListEntry() []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "home-auction-entry", NewRoomListEntry(ar))
	if err != nil {
		log.Fatalf("Template parsing error: %s", err)
	}

	return renderedMsg.Bytes()
}

func (ar *AuctionRoom) RenderNewRoomEntry() []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "appendable-auction-entry", NewRoomListEntry(ar))
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
	return &RoomManager{
		Rooms:      make(map[string]*AuctionRoom),
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
			log.Printf("Room registered: %s\n", auctionRoom.Id)
		case auctionRoom := <-rm.unregister:
			if _, ok := rm.Rooms[auctionRoom.Id]; ok {
				delete(rm.Rooms, auctionRoom.Id)
				log.Printf("Room unregistered: %s\n", auctionRoom.Id)
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

// NOTE: again, need to define this here instead of in templating due to the circular imports
type HomePage struct {
	IsAdmin     bool
	RoomManager *RoomManager
}

func NewHomePage(isAdmin bool, roomManager *RoomManager) HomePage {
	return HomePage{IsAdmin: isAdmin, RoomManager: roomManager}
}
