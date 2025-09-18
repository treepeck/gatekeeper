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

/*
Server is responsible for handling incoming events from both connected
clients and the core server.

Communication with the core server is done via an AMQP (RabbitMQ) channel, where
the server publishes client events.  It also consumes the core queue, where the
core server publishes responses.
*/
type Server struct {
	Channel    *amqp091.Channel
	Register   chan Handshake
	unregister chan string
	EventBus   chan types.MetaEvent
	clients    map[string]*client
	rooms      map[string]room
}

func NewServer(ch *amqp091.Channel) *Server {
	rooms := make(map[string]room, 1)
	rooms["hub"] = newHubRoom()

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
Run recieves events from the [Server] channels and routes them to the
corresponding handlers.
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

	session, err := h.Request.Cookie("Auth")
	if err != nil {
		http.Error(h.ResponseWriter, "Sign up/in to start playing.", http.StatusUnauthorized)
		return
	}

	req, err := http.NewRequest(
		http.MethodPost,
		os.Getenv("AUTH_URL"),
		bytes.NewReader([]byte(session.Value)),
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

	c := newClient(playerId, roomId, s.unregister, s.EventBus, conn)

	go c.read()
	go c.write()

	s.clients[playerId] = c

	r.subscribe(c)

	log.Printf("client \"%s\" registered to room \"%s\"", playerId, roomId)

	s.rooms["hub"].broadcast(types.Event{
		Action:  types.ActionClientsCounter,
		Payload: []byte(strconv.Itoa(len(s.clients))),
	})

	// Notify the core server that the client has connected to the game room.
	if c.roomId != "hub" {
		raw, err := json.Marshal(types.MetaEvent{
			Action:   types.ActionJoinRoom,
			ClientId: playerId,
			RoomId:   c.roomId,
			Payload:  nil,
		})
		if err != nil {
			log.Printf("cannot encode leave room event: %s", err)
			return
		}

		mq.Publish(s.Channel, "gate", raw)
	}
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
	}

	log.Printf("client \"%s\" unregistered from room \"%s\"", id, c.roomId)

	s.rooms["hub"].broadcast(types.Event{
		Action:  types.ActionClientsCounter,
		Payload: []byte(strconv.Itoa(len(s.clients))),
	})

	// Notify the core server that the client has disconnected.
	raw, err := json.Marshal(types.MetaEvent{
		Action:   types.ActionLeaveRoom,
		ClientId: id,
		RoomId:   c.roomId,
		Payload:  nil,
	})
	if err != nil {
		log.Printf("cannot encode leave room event: %s", err)
		return
	}

	mq.Publish(s.Channel, "gate", raw)
}

/*
handleEvent handles the incomming event.  It's a caller's responsibility to
ensure that event has a valid payload and can be handled (the room exists).
*/
func (s *Server) handleEvent(e types.MetaEvent) {
	switch e.Action {
	// Client events.

	// Chat messages are not passed to the core server since there is no point
	// in doing so.
	case types.ActionChat:
		// Hub room does not have a chat.
		if e.RoomId == "hub" {
			return
		}

		if r, exists := s.rooms[e.RoomId]; exists {
			r.broadcast(types.Event{Action: types.ActionChat, Payload: e.Payload})
		}

	case types.ActionMakeMove:
		if _, exists := s.rooms[e.RoomId]; exists {
			if raw, err := json.Marshal(e); err == nil {
				mq.Publish(s.Channel, "gate", raw)
			}
		}

	case types.ActionEnterMatchmaking:
		// Deny the request if the client is already in the game room.
		if e.RoomId != "hub" {
			return
		}

		if raw, err := json.Marshal(e); err == nil {
			mq.Publish(s.Channel, "gate", raw)
		}

	// Server events.

	case types.ActionAddRoom:
		var dto types.AddRoom
		if err := json.Unmarshal(e.Payload, &dto); err == nil {
			s.rooms[e.RoomId] = newGameRoom(dto.WhiteId, dto.BlackId)

			// Encode the redirect event.
			if raw, err := json.Marshal(types.Event{
				Action:  types.ActionRedirect,
				Payload: []byte(strconv.Quote(e.RoomId)),
			}); err == nil {
				// Send redirect only to the room subscribers.
				if c := s.clients[dto.WhiteId]; c != nil {
					c.send <- raw
				}

				if c := s.clients[dto.BlackId]; c != nil {
					c.send <- raw
				}
			}

			log.Printf("room \"%s\" added", e.RoomId)
		}

	case types.ActionRemoveRoom:
		if r, exists := s.rooms[e.RoomId]; exists {
			delete(s.rooms, e.RoomId)

			r.destroy()

			log.Printf("room \"%s\" removed", e.RoomId)
		}

	case types.ActionGameInfo:
		fallthrough
	case types.ActionCompletedMove:
		if r, exists := s.rooms[e.RoomId]; exists {
			r.broadcast(types.Event{Action: e.Action, Payload: e.Payload})
		} else {
			log.Print("room does not exist")
		}

	default:
		log.Print("recieved an event with incorrect type")
	}
}
