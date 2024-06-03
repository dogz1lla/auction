package room

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type RoomUpdatesMessage struct {
	WsClient *RoomUpdatesClient
	Bid      float64
}

type WSRoomUpdatesMessage struct {
	Headers interface{} `json:"HEADERS"`
	// TODO map uids into user names
	//BidderId string      `json:"bidderId"`
	Bid float64 `json:"bid,string"`
}

type RoomUpdatesClient struct {
	id   string
	hub  *RoomUpdatesHub
	conn *websocket.Conn
	send chan []byte
}

func ServerRoomUpdatesWs(hub *RoomUpdatesHub, c echo.Context) {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println(err)
		return
	}

	id := uuid.New().String()
	client := &RoomUpdatesClient{
		id:   id,
		hub:  hub,
		conn: conn,
		send: make(chan []byte),
	}

	client.hub.register <- client

	go client.WriteLoop()
	go client.ReadLoop()
}

func (c *RoomUpdatesClient) ReadLoop() {
	defer func() {
		c.conn.Close()
		c.hub.unregister <- c
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		// log.Printf("new ws msg %v\n", string(text))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected ws close error: %v\n", err)
			}
			break
		}
	}
}

func (c *RoomUpdatesClient) WriteLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(msg)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type RoomUpdatesHub struct {
	clients map[*RoomUpdatesClient]bool

	broadcast  chan *AuctionRoom
	register   chan *RoomUpdatesClient
	unregister chan *RoomUpdatesClient
}

func NewRoomUpdatesHub() *RoomUpdatesHub {
	return &RoomUpdatesHub{
		clients:    make(map[*RoomUpdatesClient]bool),
		broadcast:  make(chan *AuctionRoom),
		register:   make(chan *RoomUpdatesClient),
		unregister: make(chan *RoomUpdatesClient),
	}
}

func (h *RoomUpdatesHub) Run() {
	for {
		select {
		case client := <-h.register:
			// NOTE maps in go are not concurrent so use the lock (mentioned at 24:15 in the video)
			h.clients[client] = true
			log.Printf("TEST client registered: %s\n", client.id)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
				log.Printf("TEST client unregistered: %s\n", client.id)
			}
		case auctionRoom := <-h.broadcast:
			log.Printf("TEST room entry update: %v\n", auctionRoom)
			log.Printf("TEST room entry update template: %s\n", string(auctionRoom.RenderRoomListEntry()))
			// broadcast the new state
			for client := range h.clients {
				select {
				case client.send <- auctionRoom.RenderRoomListEntry():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
