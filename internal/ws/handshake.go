package ws

import (
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
Handshake represents a single HTTP request.
*/
type Handshake struct {
	Request         *http.Request
	ResponseWriter  http.ResponseWriter
	ResponseChannel chan struct{}
}
