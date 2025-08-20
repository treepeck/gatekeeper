package ws

import (
	"crypto/rand"
	"encoding/json"
	"log"

	"github.com/BelikovArtem/gatekeeper/pkg/event"
	"github.com/BelikovArtem/gatekeeper/pkg/mq"
	"github.com/gorilla/websocket"
)

/*
Gatekeeper handles client connections, disconnections and routes incomming
messages into the conrresponding rooms.
*/
type Gatekeeper struct {
	clientsCounter int
	dialer         mq.Dialer
	unregister     chan *client
	bus            chan event.ClientEvent
	Register       chan *websocket.Conn
	rooms          map[string]*room
}

/*
NewCore opens an AMQP channel, declares the exchange and creates a new Gatekeeper
instance.  The created channel will be taken by the HUB room after the exchange
declaration.
*/
func NewGatekeeper(d mq.Dialer) *Gatekeeper {
	ch, err := d.Connection.Channel()
	if err != nil {
		log.Fatalf("cannot declare a Gatekeeper channel")
	}
	err = ch.ExchangeDeclare("hub", "topic", false, true, false, false, nil)
	if err != nil {
		ch.Close()
		log.Fatalf("cannot declare an exchange: %s", err)
	}

	rooms := make(map[string]*room, 1)
	rooms["hub"] = newRoom("hub", ch)

	g := &Gatekeeper{
		dialer:     d,
		unregister: make(chan *client),
		bus:        make(chan event.ClientEvent),
		Register:   make(chan *websocket.Conn),
		rooms:      rooms,
	}

	go g.routeEvents()

	return g
}

/*
routeEvents consiquentially (one at a time) recieves incomming events from the
gatekeeper channels and forwards them to the corresponding handlers.
*/
func (g *Gatekeeper) routeEvents() {
	for {
		select {
		case conn := <-g.Register:
			g.handleRegister(conn)

		case c := <-g.unregister:
			g.handleUnregister(c)

		case e := <-g.bus:
			g.route(e)
		}
	}
}

/*
handleRegister registers a new client and broadcasts the clientCounter among
all subscribed to the "hub" room clients.
*/
func (g *Gatekeeper) handleRegister(conn *websocket.Conn) {
	g.clientsCounter++

	c := newClient(rand.Text(), g, conn)
	g.rooms["hub"].subscribe(c)

	raw := event.EncodeOrPanic(event.ServerEvent{
		Action:  event.CLIENTS_COUNTER,
		Payload: event.EncodeOrPanic(g.clientsCounter),
	})
	g.rooms["hub"].broadcast(raw)
}

/*
handleUnregister unregisters the client and broadcasts the clientCounter among
all subscribed to the "hub" room clients.
*/
func (g *Gatekeeper) handleUnregister(c *client) {
	g.clientsCounter--

	r, exists := g.rooms[c.roomId]
	if !exists {
		log.Printf("client \"%s\" is subscribed to \"%s\" which doesn't exist",
			c.id, c.roomId)
		return
	}

	r.unsubscribe(c)

	raw := event.EncodeOrPanic(event.ServerEvent{
		Action:  event.CLIENTS_COUNTER,
		Payload: event.EncodeOrPanic(g.clientsCounter),
	})
	g.rooms["hub"].broadcast(raw)
}

/*
handleCreateRoom creates a new room.  A single client cannot create or even be
subscribed to multiple rooms simultaneously.
*/
func (g *Gatekeeper) handleCreateRoom(c *client) {
	ch, err := g.dialer.OpenChannel()
	if err != nil {
		log.Printf("cannot open channel for a new room: %s", err)
		return
	}

	id := rand.Text()
	r := newRoom(id, ch)
	r.subscribe(c)

	g.rooms[id] = r
}

/*
route forwards events to the room if the room exists and the publisher is
subscribed to that room.
*/
func (g *Gatekeeper) route(e event.ClientEvent) {
	r, exists := g.rooms[e.RoomId]
	if !exists {
		log.Printf("client \"%s\" sends a message to \"%s\" which doesn't exist",
			e.ClientId, e.RoomId)
		return
	} else if _, exists := r.subs[e.ClientId]; !exists {
		log.Printf("client \"%s\" sends a message to \"%s\" but isn't subscribed",
			e.ClientId, e.RoomId)
		return
	}

	switch e.Action {
	case event.CREATE_ROOM:
		c, exists := g.rooms["hub"].subs[e.ClientId]
		if e.RoomId != "hub" || !exists {
			return
		}
		g.handleCreateRoom(c)
		// Send redirect to the room creator.
		raw := event.EncodeOrPanic(event.ServerEvent{
			Action: event.REDIRECT,
			// At this point roomId will change to the id of the created room.
			Payload: event.EncodeOrPanic(c.roomId),
		})
		c.send <- raw
	}

	raw, err := json.Marshal(e)
	if err != nil {
		log.Printf("cannot encode event from \"%s\"", e.ClientId)
		return
	}
	mq.Publish(r.channel, r.id+".out", raw)
}

/*
Destroy destroys all rooms in a gatekeeper.
*/
func (g *Gatekeeper) Destroy() {
	for _, r := range g.rooms {
		r.destroy()
	}
}
