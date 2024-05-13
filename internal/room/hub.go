package room

import (
	"bytes"
	"log"

	"github.com/dogz1lla/auction/internal/templating"
)

type Message struct {
	ClientID string
	Bid      float64
}

type WSMessage struct {
	Headers interface{} `json:"HEADERS"`
	// TODO map uids into user names
	//BidderId string      `json:"bidderId"`
	Bid float64 `json:"bid,string"`
}

type Hub struct {
	clients map[*Client]bool

	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	// TODO add a room management logic
	mockRoom := NewAuctionRoom()

	for {
		select {
		case client := <-h.register:
			// NOTE maps in go are not concurrent so use the lock (mentioned at 24:15 in the video)
			h.clients[client] = true
			log.Printf("client registered: %s\n", client.id)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
				log.Printf("client unregistered: %s\n", client.id)
			}
		case msg := <-h.broadcast:
			// log messages in the hub if necessary...
			// ...
			// update the room state
			// TODO need to choose the correct room
			mockRoom.ProcessBid(msg.ClientID, msg)

			// broadcast the new state
			for client := range h.clients {
				select {
				case client.send <- RenderAuctionState(mockRoom):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func RenderAuctionState(ar *AuctionRoom) []byte {
	tmpl := templating.NewTemplate()

	var renderedMsg bytes.Buffer
	err := tmpl.Templates.ExecuteTemplate(&renderedMsg, "auction-state", ar)
	if err != nil {
		log.Fatalf("template parsing: %s", err)
	}

	return renderedMsg.Bytes()
}
