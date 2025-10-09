package ws

import (
	"bytes"
	"context"
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
Handshake represents a single HTTP request.  ResponseChannel is used to ensure
that requests are handled sequentially, not concurrently.
*/
type Handshake struct {
	Request         *http.Request
	ResponseWriter  http.ResponseWriter
	ResponseChannel chan struct{}
}

// ctxKey is used as a context type which provides player id.
type ctxKey string

const pidKey ctxKey = "pid"

/*
Authorize is a middleware that parses sessionId from the Auth cookie and sends a
request to the core server to verify the session.  If the session is valid, the
core server returns a playerId, which is then passed to the next handler
function through the request context.

The main purpose of this function is to verify and authorize the WebSocketconnection.
*/
func Authorize(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		session, err := r.Cookie("Auth")
		if err != nil {
			http.Error(rw, "Sign up/in to start playing.", http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequest(
			http.MethodPost,
			os.Getenv("AUTH_URL"),
			bytes.NewReader([]byte(session.Value)),
		)
		if err != nil {
			// Probably the only case when NewRequest will return an error is a
			// malformed cookie value.
			http.Error(rw, "Sign up/in to start playing.", http.StatusUnauthorized)
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(rw, "Server is down or busy. Please try again later.", http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()

		raw, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != http.StatusOK {
			http.Error(rw, "Sign up/in to start playing.", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), pidKey, string(raw))
		next.ServeHTTP(rw, r.WithContext(ctx))
	}
}
