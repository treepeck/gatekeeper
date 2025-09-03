package ws

import (
	"log"
	"time"

	"github.com/treepeck/gatekeeper/pkg/event"

	"github.com/gorilla/websocket"
)

// Connection parameters.
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second
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
	// server will handle the incomming client events.
	server  *Server
	forward chan<- event.ExternalEvent
	send    chan []byte
	conn    *websocket.Conn
	// Connection delay.
	delay         int
	pingTimestamp int64
}

/*
newClient creates a new client and sets the WebSocket connection properties.
*/
func newClient(forward chan<- event.ExternalEvent, s *Server, conn *websocket.Conn) *client {
	c := &client{
		server:        s,
		forward:       forward,
		send:          make(chan []byte, 192),
		conn:          conn,
		delay:         0,
		pingTimestamp: time.Now().UnixMilli(),
	}

	// Set connection parameters.
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(c.pongHandler)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	return c
}

/*
read consequentially (one at a time) reads messages from the connection and
forwards them to the gatekeeper.
*/
func (c *client) read(id string) {
	defer c.cleanup(id)

	var e event.ExternalEvent
	for {
		err := c.conn.ReadJSON(&e)
		if err != nil {
			return
		}

		e.ClientId = id

		c.forward <- e
	}
}

/*
write consequentially (one at a time) writes messages to the connection.
Automatically sends ping messages to maintain a hearbeat.
*/
func (c *client) write() {
	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

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

			c.pingTimestamp = time.Now().UnixMilli()

			err := c.conn.WriteMessage(websocket.PingMessage, event.EncodeOrPanic(
				event.ExternalEvent{
					Action:  event.Ping,
					Payload: event.EncodeOrPanic(c.delay),
				},
			))
			if err != nil {
				log.Print(err)
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
	c.delay = int(time.Now().UnixMilli() - c.pingTimestamp)
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}

/*
cleanup closes the connection and unregisters the client from the gatekeeper.
*/
func (c *client) cleanup(id string) {
	close(c.send)
	c.conn.Close()
	c.server.unregister <- id
}
