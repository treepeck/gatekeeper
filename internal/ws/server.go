package ws

import (
	"encoding/json"
	"log"

	"github.com/rabbitmq/amqp091-go"

	"github.com/treepeck/gatekeeper/pkg/event"
	"github.com/treepeck/gatekeeper/pkg/mq"
)

/*
Server stores the subscribtion map, handles client registration, unregistration,
and routes the incomming events into the corresponding handlers.
*/
type Server struct {
	clientsCounter int
	channel        *amqp091.Channel
	register       chan *client
	unregister     chan *client
	// Incomming client events.
	ExternalBus chan event.ExternalEvent
	// Incomming server events.
	InternalBus chan event.InternalEvent
	subs        map[string]map[*client]struct{}
}

/*
NewServer initializes the [Server] fields and adds the default "hub" record
in the subscribtion map.
*/
func NewServer(ch *amqp091.Channel) *Server {
	subs := make(map[string]map[*client]struct{}, 1)
	subs["hub"] = make(map[*client]struct{})

	return &Server{
		channel:     ch,
		register:    make(chan *client),
		unregister:  make(chan *client),
		ExternalBus: make(chan event.ExternalEvent),
		InternalBus: make(chan event.InternalEvent),
		subs:        subs,
	}
}

/*
Run accepts events from the [Server] channels and calls the corresponding event
handlers.
*/
func (s *Server) Run() {
	for {
		select {
		case c := <-s.register:
			s.handleRegister(c)

		case c := <-s.unregister:
			s.handleUnregister(c)

		case e := <-s.ExternalBus:
			s.handleExternalEvent(e)

		case e := <-s.InternalBus:
			s.handleInternalEvent(e)
		}
	}
}

/*
handleRegister registers a new client and broadcasts the clientCounter among
all subscribed to the "hub" room clients.
*/
func (s *Server) handleRegister(c *client) {
	// Deny the connection if the client wants to connect to a room which does
	// not exist.
	if _, exists := s.subs[c.roomId]; !exists {
		c.conn.Close()
		return
	}

	// Add the subscribtion record.
	s.clientsCounter++
	s.subs[c.roomId][c] = struct{}{}

	// Run the client's goroutines.
	go c.read()
	go c.write()

	s.encodeAndNotify(event.ExternalEvent{
		Action:  event.ClientsCounter,
		Payload: event.EncodeOrPanic(s.clientsCounter),
	}, "hub")
}

/*
handleUnregister unregisters the client and broadcasts the clientCounter among
all subscribed to the "hub" room clients.
*/
func (s *Server) handleUnregister(c *client) {
	// Delete the subscribtion record.
	s.clientsCounter--
	delete(s.subs[c.roomId], c)

	s.encodeAndNotify(event.ExternalEvent{
		Action:  event.ClientsCounter,
		Payload: event.EncodeOrPanic(s.clientsCounter),
	}, "hub")
}

/*
handleExternalEvent accepts the incomming [ExternalEvent] and publishes it into
a gate queue.  Denies the Matchmaking event if the client isn't in the hub room.
*/
func (s *Server) handleExternalEvent(e event.ExternalEvent) {
	switch e.Action {
	case event.Matchmaking:
		// RoomId is always the room to which the client is subscribed.
		if e.RoomId != "hub" {
			return // Deny the request.
		}

	case event.Chat:
		raw, err := json.Marshal(e)
		if err != nil {
			log.Printf("cannot encode event from \"%s\"", e.ClientId)
			return
		}

		for sub := range s.subs[e.RoomId] {
			sub.send <- raw
		}
		return // No need to pass the chat message to the core server.
	}

	raw, err := json.Marshal(event.InternalEvent(e))
	if err != nil {
		log.Printf("cannot encode event from \"%s\"", e.ClientId)
		return
	}

	// Publish event with metadata.
	mq.Publish(s.channel, "gate", raw)
}

/*
handleInternalEvent accepts the incomming [InternalEvent], removes the metadata
and notifies the subscribed clients about the event.
*/
func (s *Server) handleInternalEvent(e event.InternalEvent) {
	switch e.Action {
	case event.AddRoom:
		var id string
		if err := json.Unmarshal(e.Payload, &id); err != nil {
			log.Print(err)
			return
		}

		s.subs[id] = make(map[*client]struct{})
		return

	case event.RemoveRoom:
		var id string
		if err := json.Unmarshal(e.Payload, &id); err != nil {
			log.Print(err)
			return
		}

		delete(s.subs, id)
		return
	}

	s.encodeAndNotify(event.ExternalEvent{
		Action:  e.Action,
		Payload: e.Payload,
	}, e.RoomId)
}

/*
encodeAndNotify is a helper function that encodes the specified event and
broadcasts it among all subscribed to roomId clients.
*/
func (s *Server) encodeAndNotify(e event.ExternalEvent, roomId string) error {
	// Encode event payload.
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}

	for sub := range s.subs[roomId] {
		sub.send <- raw
	}
	return nil
}
