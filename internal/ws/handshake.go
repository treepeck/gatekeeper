package ws

import (
	"bytes"
	"io"
	"net/http"
	"os"

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
HandleHandshake handles incoming WebSocket handshake requests.  It upgrades the
HTTP connection to the WebSocket protocol and stores the [client].

During the handshake, the [Server] performs authentication by sending an HTTP
POST request to the endpoint specified in the AUTH_URL environment variable.
The request body contains the session id, extracted from the client's cookie.

The authentication endpoint must validate the session by looking it up in the
session database.  If valid, the endpoint must return the corresponding player
id, which is then associated with the client connection.
*/
func HandleHandshake(rw http.ResponseWriter, r *http.Request, s *Server) {
	if len(r.Cookies()) != 1 || r.Cookies()[0].Name != "Auth" {
		http.Error(rw, "Unauthorized request.", http.StatusUnauthorized)
		return
	}

	sessionId := r.Cookies()[0].Value

	req, err := http.NewRequest(
		http.MethodPost,
		os.Getenv("AUTH_URL"),
		bytes.NewReader([]byte(sessionId)),
	)
	if err != nil {
		http.Error(rw, "Unauthorized request.", http.StatusUnauthorized)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(rw, "Unauthorized request.", http.StatusUnauthorized)
		return
	}

	playerId, err := io.ReadAll(res.Body)
	if err != nil || len(sessionId) != 32 {
		http.Error(rw, "Unauthorized request.", http.StatusBadRequest)
		return
	}
	res.Body.Close()

	roomId := r.URL.Query().Get("rid")

	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		// Upgrader writes the response, so simply return here.
		return
	}

	c := newClient(string(playerId), roomId, s, conn)

	s.register <- c
}
