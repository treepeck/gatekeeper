package ws

import (
	"time"

	"github.com/BelikovArtem/gatekeeper/pkg/event"
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

/*
client wraps a single connection and provides methods for reading, writing and
handling WebSocket messages.
*/
type client struct {
	id string
	// id of the room to which the client is subscribed.
	roomId string
	// gatekeeper will handle the incomming client events.  Nil by default.
	gatekeeper *Gatekeeper
	// send must be buffered, otherwise if the goroutine writes to it but the
	// client drops the connection, the goroutine will wait forever.
	send chan []byte
	conn *websocket.Conn
	// is WebSocket connection alive.
	isAlive bool
}

func newClient(id, roomId string, conn *websocket.Conn) *client {
	c := &client{
		id:      id,
		roomId:  roomId,
		send:    make(chan []byte, 192),
		conn:    conn,
		isAlive: true,
	}

	// Set connection parameters.
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(c.pongHandler)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	go c.read()
	go c.write()

	return c
}

/*
read consequentially (one at a time) reads messages from the connection and
forwards them to the gatekeeper.
*/
func (c *client) read() {
	defer func() {
		c.cleanup()
	}()

	var e event.ClientEvent
	for {
		err := c.conn.ReadJSON(&e)
		if err != nil {
			return
		}

		e.RoomId = c.roomId
		e.ClientId = c.id

		c.gatekeeper.bus <- e
	}
}

/*
write consequentially (one at a time) writes messages to the connection.
Automatically sends ping messages to maintain a hearbeat.
*/
func (c *client) write() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		c.cleanup()
	}()

	for {
		select {
		case raw, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.conn.WriteMessage(websocket.BinaryMessage, raw)
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

/*
pongHandler handles the incomming pong messages to maintain a heartbeat.

Sending ping and pong messages is necessary because without it the connections
are interrupted after about 2 minutes of no message sending from the client.
*/
func (c *client) pongHandler(appData string) error {
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}

/*
cleanup closes the connection and unregisters the client from the gatekeeper.
*/
func (c *client) cleanup() {
	if c.isAlive {
		c.isAlive = false
		c.conn.Close()

		if c.gatekeeper != nil {
			c.gatekeeper.unregister <- c
		}

		close(c.send)
	}
}
