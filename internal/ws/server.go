package ws

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/treepeck/gatekeeper/pkg/event"
	"github.com/treepeck/gatekeeper/pkg/mq"

	"github.com/rabbitmq/amqp091-go"
)

type Server struct {
	Channel       *amqp091.Channel
	Register      chan Handshake
	unregister    chan string
	externalEvent chan event.ExternalEvent
	InternalEvent chan event.InternalEvent
	clients       map[string]*client
	rooms         map[string]room
}

func NewServer(ch *amqp091.Channel) *Server {
	rooms := make(map[string]room, 1)
	rooms["hub"] = newHubRoom()

	return &Server{
		Channel:       ch,
		Register:      make(chan Handshake),
		unregister:    make(chan string),
		externalEvent: make(chan event.ExternalEvent),
		InternalEvent: make(chan event.InternalEvent),
		clients:       make(map[string]*client),
		rooms:         rooms,
	}
}

/*
Run recieves events from the [Server] channels and calls the corresponding
handlers.
*/
func (s *Server) Run() {
	for {
		select {
		case h := <-s.Register:
			s.handleRegister(h)

		case id := <-s.unregister:
			s.handleUnregister(id)

		case e := <-s.externalEvent:
			s.handleExternalEvent(e)

		case e := <-s.InternalEvent:
			s.handleInternalEvent(e)
		}
	}
}

/*
handleRegister handles the incomming WebSocket [Handshake] requests.
*/
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

	s.rooms["hub"].broadcast(event.ExternalEvent{
		Action:  event.ClientsCounter,
		Payload: event.EncodeOrPanic(len(s.clients)),
	})
}

/*
handleUnregister unregisters the client.
*/
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

	s.rooms["hub"].broadcast(event.ExternalEvent{
		Action:  event.ClientsCounter,
		Payload: event.EncodeOrPanic(len(s.clients)),
	})
}

/*
handleExternalEvent handles a [ExternalEvent] recieved from the connected client.
*/
func (s *Server) handleExternalEvent(e event.ExternalEvent) {
	// Deny the request if the client is not registered.
	c, exists := s.clients[e.ClientId]
	if !exists {
		log.Printf("client \"%s\" sent a message but is not registered", e.ClientId)
		return
	}

	// Deny the request if the room does not exist.
	r, exists := s.rooms[c.roomId]
	if !exists {
		log.Printf("room \"%s\" does not exist", c.roomId)
		return
	}

	switch e.Action {
	// Chat messages are not passed to the core server since there is no point
	// in doing so.
	case event.Chat:
		r.broadcast(e)
		return

	default:
		mq.Publish(s.Channel, "gate", event.EncodeOrPanic(event.InternalEvent{
			Action:   e.Action,
			Payload:  e.Payload,
			ClientId: e.ClientId,
			RoomId:   c.roomId,
		}))
	}
}

/*
handleInternalEvent handles a [InternalEvent] recieved from the core server.
*/
func (s *Server) handleInternalEvent(e event.InternalEvent) {
	switch e.Action {
	// Clients will be automatically unsubscribed from the HUB room after
	// redirect.
	case event.AddRoom:
		var p event.AddRoomPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			log.Printf("cannot decode AddRoomPayload: %s", err)
			return
		}

		r := newGameRoom()
		s.rooms[p.Id] = r

		// Send redirect to the players.
		redir := event.ExternalEvent{
			Action:  event.Redirect,
			Payload: event.EncodeOrPanic(p.Id),
		}
		s.clients[p.WhiteId].send <- redir
		s.clients[p.BlackId].send <- redir

	case event.RemoveRoom:
		var roomId string
		if err := json.Unmarshal(e.Payload, &roomId); err != nil {
			log.Printf("cannot decode room id: %s", err)
			return
		}
		delete(s.rooms, roomId)
	}
}
