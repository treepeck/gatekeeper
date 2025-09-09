package ws

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/treepeck/gatekeeper/pkg/mq"
	"github.com/treepeck/gatekeeper/pkg/types"

	"github.com/rabbitmq/amqp091-go"
)

type Server struct {
	Channel       *amqp091.Channel
	Register      chan Handshake
	unregister    chan string
	externalEvent chan types.ExternalEvent
	InternalEvent chan types.InternalEvent
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
		externalEvent: make(chan types.ExternalEvent),
		InternalEvent: make(chan types.InternalEvent),
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

	raw, err = json.Marshal(len(s.clients))
	if err != nil {
		log.Printf("cannot encode clients counter: %s", err)
		return
	}
	s.rooms["hub"].broadcast(types.ExternalEvent{
		Action:  types.ActionClientsCounter,
		Payload: raw,
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

	raw, err := json.Marshal(len(s.clients))
	if err != nil {
		log.Printf("cannot encode clients counter: %s", err)
		return
	}
	s.rooms["hub"].broadcast(types.ExternalEvent{
		Action:  types.ActionClientsCounter,
		Payload: raw,
	})
}

/*
handleExternalEvent handles a [ExternalEvent] recieved from the connected client.
*/
func (s *Server) handleExternalEvent(e types.ExternalEvent) {
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
	case types.ActionChat:
		r.broadcast(e)
		return

	default:
		raw, err := json.Marshal(types.InternalEvent{
			Action:   e.Action,
			Payload:  e.Payload,
			ClientId: e.ClientId,
			RoomId:   c.roomId,
		})
		if err != nil {
			log.Printf("cannot encode internal event: %s", err)
			return
		}
		mq.Publish(s.Channel, "gate", raw)
	}
}

/*
handleInternalEvent handles a [InternalEvent] recieved from the core server.
*/
func (s *Server) handleInternalEvent(e types.InternalEvent) {
	switch e.Action {
	// Clients will be automatically unsubscribed from the HUB room after
	// redirect.
	case types.ActionAddRoom:
		var dto types.AddRoom
		if err := json.Unmarshal(e.Payload, &dto); err != nil {
			log.Printf("cannot decode AddRoomPayload: %s", err)
			return
		}

		r := newGameRoom()
		s.rooms[dto.Id] = r

		// Send redirect to the players.
		raw, err := json.Marshal(types.ExternalEvent{
			Action:  types.ActionRedirect,
			Payload: []byte(dto.Id),
		})
		if err != nil {
			log.Printf("cannot encode external event: %s", err)
			return
		}
		s.clients[dto.WhiteId].send <- raw
		s.clients[dto.BlackId].send <- raw

	case types.ActionRemoveRoom:
		var roomId string
		if err := json.Unmarshal(e.Payload, &roomId); err != nil {
			log.Printf("cannot decode room id: %s", err)
			return
		}
		delete(s.rooms, roomId)

	case types.ActionCompletedMove:
		raw, err := json.Marshal(types.ExternalEvent{
			Action:  types.ActionCompletedMove,
			Payload: e.Payload,
		})
		if err != nil {
			log.Printf("cannot marshal external event: %s", err)
			return
		}

		for _, c := range s.clients {
			c.send <- raw
		}
	}
}
