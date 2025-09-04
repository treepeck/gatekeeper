package ws

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/treepeck/gatekeeper/pkg/event"
)

type Server struct {
	Register      chan Handshake
	unregister    chan string
	externalEvent chan event.ExternalEvent
	clients       map[string]*client
	rooms         map[string]room
}

func NewServer() *Server {
	rooms := make(map[string]room, 1)
	rooms["hub"] = newHubRoom()

	return &Server{
		Register:      make(chan Handshake),
		unregister:    make(chan string),
		externalEvent: make(chan event.ExternalEvent),
		clients:       make(map[string]*client),
		rooms:         rooms,
	}
}

func (s *Server) Run() {
	for {
		select {
		case h := <-s.Register:
			s.handleRegister(h)

		case id := <-s.unregister:
			s.handleUnregister(id)

		case e := <-s.externalEvent:
			s.handleExternalEvent(e)
		}
	}
}

func (s *Server) handleRegister(h Handshake) {
	defer func() { h.ResponseChannel <- struct{}{} }()

	roomId := h.Request.URL.Query().Get("rid")
	r, exists := s.rooms[roomId]
	if !exists {
		http.Error(h.ResponseWriter, "The requested room not found.", http.StatusNotFound)
		return
	}

	cookie, err := h.Request.Cookie("Auth")
	if err != nil {
		http.Error(h.ResponseWriter, "Sign up/in to start playing.", http.StatusUnauthorized)
		return
	}

	sessionId := cookie.Value

	req, err := http.NewRequest(
		http.MethodPost,
		os.Getenv("AUTH_URL"),
		bytes.NewReader([]byte(sessionId)),
	)
	if err != nil {
		http.Error(h.ResponseWriter, "Please try again later.", http.StatusInternalServerError)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(h.ResponseWriter, "Core server is busy or down.", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != http.StatusOK {
		http.Error(h.ResponseWriter, "Sign up/in to start playing.", http.StatusUnauthorized)
		return
	}
	playerId := string(raw)

	conn, err := upgrader.Upgrade(h.ResponseWriter, h.Request, nil)
	if err != nil {
		return
	}

	c := newClient(roomId, s.externalEvent, s, conn)

	go c.read(playerId)
	go c.write()

	s.clients[playerId] = c

	r.subscribe(playerId, c)

	log.Printf("client \"%s\" registered to room \"%s\"", playerId, roomId)
}

func (s *Server) handleUnregister(id string) {
	c, exists := s.clients[id]
	if !exists {
		log.Printf("client \"%s\" does not exist", id)
		return
	}

	delete(s.clients, id)

	if r, exists := s.rooms[c.roomId]; exists {
		r.unsubscribe(id)
	} else {
		log.Printf("client subscribed to room \"%s\" which does not exist", c.roomId)
	}

	log.Printf("client \"%s\" unregistered from room \"%s\"", id, c.roomId)
}

func (s *Server) handleExternalEvent(e event.ExternalEvent) {
	log.Print(e)
}
