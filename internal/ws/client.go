package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

// Connection parameters.
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

type client struct {
	id         string
	roomId     string
	gatekeeper *Gatekeeper
	send       chan ServerEvent
	conn       *websocket.Conn
	isAlive    bool
}

func newClient(id string, g *Gatekeeper, conn *websocket.Conn) *client {
	c := &client{
		id:         id,
		gatekeeper: g,
		send:       make(chan ServerEvent),
		conn:       conn,
		isAlive:    true,
	}

	// Set connection parameters.
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(c.pongHandler)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	go c.read()
	go c.write()

	return c
}

func (c *client) read() {
	defer func() {
		c.cleanup()
	}()

	for {
		var e ClientEvent
		err := c.conn.ReadJSON(&e)
		if err != nil {
			return
		}

		// Forbit the client to create rooms when it is already in the game.
		// if e.Act == create_topic || c.topicId != "hub" {
		// 	continue
		// }
		e.RoomId = c.roomId
		e.ClientId = c.id

		c.gatekeeper.bus <- e
	}
}

func (c *client) write() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		c.cleanup()
	}()

	for {
		select {
		case body, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.conn.WriteJSON(body)
			if err != nil {
				return
			}

		// Send ping messages periodically.
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}

func (c *client) pongHandler(appData string) error {
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *client) cleanup() {
	if c.isAlive {
		c.isAlive = false
		c.conn.Close()
		c.gatekeeper.unregister <- c
		close(c.send)
	}
}
