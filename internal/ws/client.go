package ws

import (
	"time"

	"github.com/treepeck/gatekeeper/pkg/types"

	"github.com/gorilla/websocket"
)

// Connection parameters.
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 7 * time.Second
	// Send pings to peer with this period.  Must be less than pongWait.
	pingPeriod = 3 * time.Second
	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

/*
client wraps a single connection and provides methods for reading, writing and
handling WebSocket messages.
*/
type client struct {
	// Timestamp when the last ping event was sent to measure response delay.
	pingTimestamp time.Time
	// Id of the room to which the client is subscribed.  Multiple subscribtions
	// from a single client are prohibited.
	roomId string
	// server will handle the incomming client events.
	server  *Server
	forward chan<- types.Event
	// send must recieve raw bytes to avoid expensive JSON encoding for each
	// client.
	send chan []byte
	conn *websocket.Conn
	// Network delay.
	delay int
	// New ping event must be sent only when the client responses to the
	// previous one.  Otherwise the delay cannot be correctly measured.
	hasAnsweredPing bool
}

/*
newClient creates a new client and sets the WebSocket connection properties.
*/
func newClient(
	roomId string,
	forward chan<- types.Event,
	s *Server,
	conn *websocket.Conn,
) *client {
	now := time.Now()

	c := &client{
		server:        s,
		roomId:        roomId,
		forward:       forward,
		send:          make(chan []byte, 192),
		conn:          conn,
		delay:         0,
		pingTimestamp: now,
		// Must be true to be able to send a first ping message.
		hasAnsweredPing: true,
	}

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(now.Add(pongWait))

	return c
}

/*
read consequentially (one at a time) reads messages from the connection and
forwards them to the gatekeeper.
*/
func (c *client) read(id string) {
	defer c.cleanup(id)

	var e types.Event
	for {
		err := c.conn.ReadJSON(&e)
		if err != nil {
			return
		}

		// Validate and add metadata before forwarding to the server.
		switch e.Action {
		case types.ActionPong:
			c.handlePong()

		case types.ActionChat:
			p, ok := e.Payload.(types.Chat)
			if !ok || c.roomId == "hub" {
				goto malformed
			}
			p.ClientId = id
			p.RoomId = c.roomId
			c.forward <- types.Event{Action: e.Action, Payload: p}

		case types.ActionMakeMove:
			p, ok := e.Payload.(types.MakeMove)
			if !ok || c.roomId == "hub" {
				goto malformed
			}
			p.ClientId = id
			p.RoomId = c.roomId
			c.forward <- types.Event{Action: e.Action, Payload: p}

		case types.ActionEnterMatchmaking:
			p, ok := e.Payload.(types.EnterMatchmaking)
			if !ok || c.roomId != "hub" {
				goto malformed
			}
			// Forward event with metadata.
			p.ClientId = id
			c.forward <- types.Event{Action: e.Action, Payload: p}
		}
		continue

	malformed:
		// Close the connection if the client sent a malformed event.
		c.send <- []byte("Server recieved a malformed message. The connection will be closed.")
		return
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

			if err := c.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}

		// Send ping messages periodically.
		case <-pingTicker.C:
			// Send a new ping event only if the client has already ansered to
			// the prevous one.
			if !c.hasAnsweredPing {
				continue
			}

			now := time.Now()
			c.conn.SetWriteDeadline(now.Add(writeWait))

			c.pingTimestamp = now

			if err := c.conn.WriteJSON(types.Event{
				Action:  types.ActionPing,
				Payload: c.delay,
			}); err != nil {
				return
			}
			c.hasAnsweredPing = false
		}
	}
}

/*
handlePong handles the incomming pong messages to maintain a heartbeat.

Sending ping and pong messages is necessary because without it the connections
are interrupted after about 2 minutes of no message sending from the client.

Sets the delay value to the time elapsed since the last ping was sent.  This
helps determine an up-to-date network delay value, which will be subtracted from
the player's clock to provide a fairer gameplay experience.
*/
func (c *client) handlePong() error {
	// Handle pong events only when the client has a pending ping event.
	if !c.hasAnsweredPing {
		c.hasAnsweredPing = true
		c.delay = int(time.Since(c.pingTimestamp).Milliseconds())
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	}
	return nil
}

/*
cleanup closes the connection and unregisters the client from the gatekeeper.
*/
func (c *client) cleanup(id string) {
	close(c.send)
	c.conn.Close()
	c.server.unregister <- id
}
