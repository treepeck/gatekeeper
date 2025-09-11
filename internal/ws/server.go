package ws

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/treepeck/gatekeeper/pkg/mq"
	"github.com/treepeck/gatekeeper/pkg/types"

	"github.com/rabbitmq/amqp091-go"
)

type Server struct {
	Channel    *amqp091.Channel
	Register   chan Handshake
	unregister chan string
	EventBus   chan types.MetaEvent
	clients    map[string]*client
	rooms      map[string]*room
}

func NewServer(ch *amqp091.Channel) *Server {
	rooms := make(map[string]*room, 1)
	rooms["hub"] = newRoom()

	return &Server{
		Channel:    ch,
		Register:   make(chan Handshake),
		unregister: make(chan string),
		EventBus:   make(chan types.MetaEvent),
		clients:    make(map[string]*client),
		rooms:      rooms,
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

		case e := <-s.EventBus:
			s.handleEvent(e)
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

	c := newClient(roomId, s.EventBus, s, conn)

	go c.read(playerId)
	go c.write()

	s.clients[playerId] = c

	r.subscribe(playerId, c)

	log.Printf("client \"%s\" registered to room \"%s\"", playerId, roomId)

	s.rooms["hub"].broadcast(types.Event{
		Action:  types.ActionClientsCounter,
		Payload: []byte(strconv.Itoa(len(s.clients))),
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

	s.rooms["hub"].broadcast(types.Event{
		Action:  types.ActionClientsCounter,
		Payload: []byte(strconv.Itoa(len(s.clients))),
	})
}

/*
handleEvent handles the incomming event.  It's a caller's responsibility to ensure
that event has a valid payload and can be handled (the room exists).
*/
func (s *Server) handleEvent(e types.MetaEvent) {
	switch e.Action {
	// Client events.

	// Chat messages are not passed to the core server since there is no point
	// in doing so.
	case types.ActionChat:
		if r, exists := s.rooms[e.RoomId]; exists {
			r.broadcast(types.Event{Action: types.ActionChat, Payload: e.Payload})
		}

	case types.ActionMakeMove:
		if _, exists := s.rooms[e.RoomId]; exists {
			raw, err := json.Marshal(e)
			if err != nil {
				log.Printf("cannot encode make move event: %s", err)
				return
			}
			mq.Publish(s.Channel, "gate", raw)
		}

	case types.ActionEnterMatchmaking:
		raw, err := json.Marshal(e)
		if err != nil {
			log.Printf("cannot encode enter matchmaking event: %s", err)
			return
		}
		mq.Publish(s.Channel, "gate", raw)

	// Server events.

	case types.ActionAddRoom:
		s.rooms[e.RoomId] = newRoom()
		log.Printf("room \"%s\" added", e.RoomId)

		s.rooms["hub"].broadcast(types.Event{
			Action:  types.ActionAddRoom,
			Payload: e.Payload,
		})

	case types.ActionGameInfo:
		if r, exists := s.rooms[e.RoomId]; exists {
			r.broadcast(types.Event{Action: e.Action, Payload: e.Payload})
		}

	case types.ActionCompletedMove:
		if r, exists := s.rooms[e.RoomId]; exists {
			r.broadcast(types.Event{Action: e.Action, Payload: e.Payload})
		}

	default:
		log.Print("recieved an event with incorrect type")
	}
}
