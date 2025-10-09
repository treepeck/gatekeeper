package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/rabbitmq/amqp091-go"

	"github.com/treepeck/gatekeeper/pkg/mq"
	"github.com/treepeck/gatekeeper/pkg/types"
)

/*
Central point within the system. Handles connections and disconnections, stores
active rooms and clients, handles events from both clients and core server.
*/
type Server struct {
	channel        *amqp091.Channel
	Register       chan Handshake
	unregister     chan *client
	request        chan types.ClientEvent
	Response       chan []byte
	rooms          map[string]room
	clientsCounter int
}

func NewServer(ch *amqp091.Channel) *Server {
	// Add the default hub room.
	rooms := make(map[string]room)
	rooms["hub"] = newHubRoom()

	return &Server{
		channel:    ch,
		Register:   make(chan Handshake),
		unregister: make(chan *client),
		request:    make(chan types.ClientEvent),
		Response:   make(chan []byte),
		rooms:      rooms,
	}
}

func (s *Server) Run() {
	for {
		select {
		case h := <-s.Register:
			s.handleHandshake(h)

		case c := <-s.unregister:
			s.handleUnregister(c)

		case e := <-s.request:
			s.handleRequest(e)

		case raw := <-s.Response:
			s.handleResponse(raw)
		}
	}
}

func (s *Server) handleRequest(e types.ClientEvent) {
	r, exists := s.rooms[e.RoomId]
	if !exists {
		log.Printf("client \"%s\" send even to non existing room \"%s\"",
			e.ClientId, e.RoomId)
		return
	}

	switch e.Action {
	// Chat messages are not passed to the core server.
	case types.ActionChat:
		r.broadcast(types.ExternalEvent{Action: e.Action, Payload: e.Payload})
		return

	case types.ActionEnterMatchmaking:
		// Deny the request if the clients is already in the game room.
		if e.RoomId != "hub" {
			return
		}
	}

	// Publish event to the core server.
	raw, err := json.Marshal(e)
	if err != nil {
		log.Printf("cannot encode request event: %s", err)
		return
	}
	mq.Publish(s.channel, "gate", raw)
}

func (s *Server) handleResponse(raw []byte) {
	var e types.ServerEvent
	if err := json.Unmarshal(raw, &e); err != nil {
		log.Printf("cannot decode respose: %s", err)
		return
	}

	switch e.Action {
	case types.ActionAddRoom:
		var dto types.AddRoom
		if err := json.Unmarshal(e.Payload, &dto); err == nil {
			r := newGameRoom(dto.WhiteId, dto.BlackId)
			s.rooms[e.RoomId] = r

			hub := s.rooms["hub"]

			// Encode the redirect.
			raw, err := json.Marshal(types.ExternalEvent{
				Action:  types.ActionRedirect,
				Payload: []byte(strconv.Quote(e.RoomId)),
			})
			if err != nil {
				log.Printf("cannot encode redirect: %s", err)
				return
			}

			// Subscribe clients to the room.
			if c := hub.getSubscriber(dto.WhiteId); c != nil {
				c.send <- raw
			}
			if c := hub.getSubscriber(dto.BlackId); c != nil {
				c.send <- raw
			}
		}
	}
}

/*
handleHandshake sequentially registers new WebSocket clients.
*/
func (s *Server) handleHandshake(h Handshake) {
	defer func() { h.ResponseChannel <- struct{}{} }()

	playerId := h.Request.Context().Value(pidKey).(string)
	roomId := h.Request.URL.Query().Get("rid")

	// Deny the request if the specified room doesn't exist or the client is
	// already registered.
	r, exists := s.rooms[roomId]
	if !exists {
		http.Error(h.ResponseWriter, "The requested room doesn't exist.", http.StatusNotFound)
		return
	} else if r.getSubscriber(playerId) != nil {
		http.Error(h.ResponseWriter, "Multiple connections from a single peer is prohibited. Please close the previous tabs and refresh the page to reconnect.", http.StatusConflict)
		return
	}

	conn, err := upgrader.Upgrade(h.ResponseWriter, h.Request, nil)
	if err != nil {
		// The upgrader writes the response, so simply return here.
		return
	}

	c := newClient(playerId, roomId, s.unregister, s.request, conn)

	go c.read()
	go c.write()

	r.subscribe(c)

	s.clientsCounter++

	s.rooms["hub"].broadcast(types.ExternalEvent{
		Action:  types.ActionClientsCounter,
		Payload: []byte(strconv.Itoa(s.clientsCounter)),
	})

	if roomId != "hub" {
		raw, err := json.Marshal(types.ClientEvent{
			Action:   types.ActionLeaveRoom,
			ClientId: playerId,
			RoomId:   roomId,
			Payload:  nil,
		})
		if err != nil {
			log.Printf("cannot encode request: %s", err)
			return
		}

		mq.Publish(s.channel, "gate", raw)
	}
}

func (s *Server) handleUnregister(c *client) {
	r, exists := s.rooms[c.roomId]
	if !exists {
		log.Printf("client \"%s\" subscribed to non-existing room \"%s\"",
			c.id, c.roomId)
		return
	}

	s.clientsCounter--

	r.unsubscribe(c.id)

	s.rooms["hub"].broadcast(types.ExternalEvent{
		Action:  types.ActionClientsCounter,
		Payload: []byte(strconv.Itoa(s.clientsCounter)),
	})

	// Notify the core server that the client has disconnected.
	raw, err := json.Marshal(types.ClientEvent{
		Action:   types.ActionLeaveRoom,
		ClientId: c.id,
		RoomId:   c.roomId,
		Payload:  nil,
	})
	if err != nil {
		log.Printf("cannot encode request: %s", err)
		return
	}

	mq.Publish(s.channel, "gate", raw)
}
