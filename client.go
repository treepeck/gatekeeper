package main

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
	id      string
	topicId string
	hub     *Hub
	send    chan event
	conn    *websocket.Conn
	isAlive bool
}

func newClient(id, topicId string, h *Hub, conn *websocket.Conn) *client {
	c := &client{
		id:      id,
		topicId: topicId,
		hub:     h,
		send:    make(chan event),
		conn:    conn,
		isAlive: true,
	}

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(c.pongHandler)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	go c.read()
	go c.write()

	return c
}

func (c *client) read() {
	defer c.cleanup()

	for {
		var e event
		err := c.conn.ReadJSON(&e)
		if err != nil {
			return
		}

		c.hub.route <- e
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
		c.hub.unregister <- c
	}
}
