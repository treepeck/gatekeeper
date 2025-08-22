package ws

import (
	"crypto/rand"
	"net/http"

	"github.com/gorilla/websocket"
)

/*
upgrader is used to establish a WebSocket connection.  It is safe for concurrent
use.
*/
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

/*
HandleHandshake handles incomming WebSocket requests, upgrades the HTTP
connection to the WebSocket protocol and registers the client to the [Server].
*/
func HandleHandshake(rw http.ResponseWriter, r *http.Request, s *Server) {
	roomId := r.URL.Query().Get("rid")

	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		// Upgrader writes the response, so simply return here.
		return
	}

	c := newClient(rand.Text(), roomId, s, conn)

	s.register <- c
}
