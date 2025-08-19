package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader is used to establish a WebSocket connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

/*
HandleHandshake handles incomming WebSocket handshake-requests, upgrades the
HTTP connection to the WebSocket protocol and registers the client in the
Gatekeeper.
*/
func HandleHandshake(rw http.ResponseWriter, r *http.Request, g *Gatekeeper) {
	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		return
	}

	g.Register <- conn
}
