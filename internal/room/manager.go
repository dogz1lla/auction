package room

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

const (
	timefmt = "2006-01-02T15:04"
)

type RoomUpdatesMessage struct {
	WsClient *RoomUpdatesClient
	ClosesAt time.Time
}

type WSRoomUpdatesMessage struct {
	Headers interface{} `json:"HEADERS"`
	// TODO map uids into user names
	//BidderId string      `json:"bidderId"`
	ClosesAt string `json:"ClosesAt"`
}

type RoomUpdatesClient struct {
	id   string
	hub  *RoomUpdatesHub
	conn *websocket.Conn
	send chan []byte
}

func ServerRoomUpdatesWs(hub *RoomUpdatesHub, roomManager *RoomManager, c echo.Context) {
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
	go client.ReadLoop(roomManager)
}

func (c *RoomUpdatesClient) ReadLoop(roomManager *RoomManager) {
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
		_, text, err := c.conn.ReadMessage()
		// log.Printf("new ws msg %v\n", string(text))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected ws close error: %v\n", err)
			}
			break
		}

		// NOTE: if there is another functionality in the future that is delivered from client
		// (like eg force closing an auction prematurely) then need to add other fields to the
		// WSRoomUpdatesMessage and then check which fields are not nil to determine which type of
		// message it is
		msg := &WSRoomUpdatesMessage{}
		reader := bytes.NewReader(text)
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(msg); err != nil {
			log.Printf("Json decoding error: %v\n", err)
		}

		if msg.ClosesAt != "" {
			closesAt, err := time.Parse(timefmt, msg.ClosesAt)
			if err != nil {
				log.Printf("time parsing error: fmt=%s, to parse=%s, err: %s", timefmt, msg.ClosesAt, err.Error())
			}
			room := roomManager.CreateAuction(closesAt)
			c.hub.newRoom <- room
			go func() {
				expiresAfter := GetMillisTill(closesAt)
				// we have a room that is already expired -> handle it right away
				if expiresAfter <= 0 {
					c.hub.roomExpired <- room
					return
				}
				// time.Duration is in nanoseconds
				expirationClock := time.After(time.Duration(1_000_000 * expiresAfter))
				for {
					select {
					case <-expirationClock:
						c.hub.roomExpired <- room
					}
				}
			}()
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

	// TODO: potentially useful to rename the broadcast chan
	broadcast   chan *AuctionRoom
	newRoom     chan *AuctionRoom
	roomExpired chan *AuctionRoom
	register    chan *RoomUpdatesClient
	unregister  chan *RoomUpdatesClient
}

func NewRoomUpdatesHub() *RoomUpdatesHub {
	return &RoomUpdatesHub{
		clients:     make(map[*RoomUpdatesClient]bool),
		broadcast:   make(chan *AuctionRoom),
		newRoom:     make(chan *AuctionRoom),
		roomExpired: make(chan *AuctionRoom),
		register:    make(chan *RoomUpdatesClient),
		unregister:  make(chan *RoomUpdatesClient),
	}
}

func (h *RoomUpdatesHub) Run() {
	for {
		select {
		case client := <-h.register:
			// NOTE maps in go are not concurrent so use the lock (mentioned at 24:15 in the video)
			h.clients[client] = true
			log.Printf("client registered (room updates): %s\n", client.id)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
				log.Printf("client unregistered (room updates): %s\n", client.id)
			}
		case auctionRoom := <-h.broadcast:
			// broadcast the new state
			for client := range h.clients {
				select {
				case client.send <- auctionRoom.RenderRoomListEntry():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case newRoom := <-h.newRoom:
			// broadcast the new room
			for client := range h.clients {
				select {
				case client.send <- newRoom.RenderNewRoomEntry():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case expiredRoom := <-h.roomExpired:
			// broadcast the new room
			for client := range h.clients {
				select {
				case client.send <- expiredRoom.RenderExpiredRoomEntry():
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
