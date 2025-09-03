package ws

import (
	"crypto/rand"
	"log"

	"github.com/treepeck/gatekeeper/pkg/event"
)

type Server struct {
	Register      chan Handshake
	unregister    chan string
	externalEvent chan event.ExternalEvent
	clients       map[string]*client
}

func NewServer() *Server {
	return &Server{
		Register:      make(chan Handshake),
		unregister:    make(chan string),
		externalEvent: make(chan event.ExternalEvent),
		clients:       make(map[string]*client),
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

	conn, err := upgrader.Upgrade(h.ResponseWriter, h.Request, nil)
	if err != nil {
		return
	}

	id := rand.Text()
	c := newClient(s.externalEvent, s, conn)

	go c.read(id)
	go c.write()

	s.clients[id] = c

	log.Printf("client \"%s\" registered", id)
}

func (s *Server) handleUnregister(id string) {
	delete(s.clients, id)
	log.Printf("client \"%s\" unregistered", id)
}

func (s *Server) handleExternalEvent(e event.ExternalEvent) {
	log.Print(e)
}
