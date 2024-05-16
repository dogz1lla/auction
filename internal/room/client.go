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

type Client struct {
	id   string
	hub  *Hub
	room *AuctionRoom
	conn *websocket.Conn
	send chan []byte
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func ServerWs(hub *Hub, roomManager *RoomManager, c echo.Context, userName, roomId string) {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println(err)
		return
	}

	room, err := roomManager.GetRoomById(roomId)
	if err != nil {
		log.Println(err)
		return
	}

	id := uuid.New()
	client := &Client{
		id:   id.String(),
		hub:  hub,
		room: room,
		conn: conn,
		send: make(chan []byte),
	}

	client.hub.register <- client

	go client.WriteLoop()
	go client.ReadLoop()
}

func (c *Client) ReadLoop() {
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

		msg := &WSMessage{}
		reader := bytes.NewReader(text)
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(msg); err != nil {
			log.Printf("Json decoding error: %v\n", err)
		}
		c.hub.broadcast <- &Message{WsClient: c, Bid: msg.Bid}
	}
}

func (c *Client) WriteLoop() {
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

			// n := len(c.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(msg)
			// }

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
