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
	CurrentBidder *users.User
	CurrentBid    float64
	// this will help render the countdown both in user's list and in the auction view
	ClosesAt time.Time
}

func NewAuctionRoom() *AuctionRoom {
	id := uuid.New()
	return &AuctionRoom{
		Id:            id.String(),
		CurrentBidder: users.NewUser("none"),
		CurrentBid:    0.0,
		// ClosesAt:      0,
	}
}

func (ar *AuctionRoom) ProcessBid(userName string, msg *Message) error {
	if time.Now().UTC().After(ar.ClosesAt) {
		errorMsg := fmt.Sprintf("Room %s has expired! No new bids allowed!", ar.Id)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}
	bid := msg.Bid
	if bid > ar.CurrentBid {
		ar.CurrentBid = bid
		ar.CurrentBidder = msg.WsClient.user
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
	User      *users.User
	Room      *AuctionRoom
	ExpiresIn int64
}

func NewRoomListEntry(user *users.User, room *AuctionRoom) *RoomListEntry {
	return &RoomListEntry{
		User:      user,
		Room:      room,
		ExpiresIn: GetMillisTill(room.ClosesAt),
	}
}

func (ar *AuctionRoom) RenderRoomListEntry(user *users.User) []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "home-auction-entry", NewRoomListEntry(user, ar))
	if err != nil {
		log.Fatalf("Template parsing error: %s", err)
	}

	return renderedMsg.Bytes()
}

func (ar *AuctionRoom) RenderNewRoomEntry(user *users.User) []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "appendable-auction-entry", NewRoomListEntry(user, ar))
	if err != nil {
		log.Fatalf("Template parsing error: %s", err)
	}

	return renderedMsg.Bytes()
}

func (ar *AuctionRoom) RenderExpiredRoomEntry(user *users.User) []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "expired-auction-entry", NewRoomListEntry(user, ar))
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

func NewAuctionPage(ar *AuctionRoom, user *users.User) *AuctionPage {
	return &AuctionPage{
		User:       user,
		Room:       ar,
		Expiration: GetMillisTill(ar.ClosesAt),
	}
}

// NOTE: again, need to define this here instead of in templating due to the circular imports
type HomePage struct {
	User *users.User
	//RoomManager *RoomManager
	RoomEntries []*RoomListEntry
}

func NewHomePage(user *users.User, roomManager *RoomManager) HomePage {
	//return HomePage{IsAdmin: isAdmin, RoomManager: roomManager}
	roomEntries := make([]*RoomListEntry, 0)
	for _, room := range roomManager.Rooms {
		roomEntries = append(roomEntries, NewRoomListEntry(user, room))
	}
	// TODO: add a check that the user exists in the db
	return HomePage{User: user, RoomEntries: roomEntries}
}
