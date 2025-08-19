package ws

import (
	"crypto/rand"
	"log"

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
	bus            chan ClientEvent
	Register       chan *websocket.Conn
	rooms          map[string]*room
}

func NewGatekeeper(d mq.Dialer) *Gatekeeper {
	ch, err := d.Connection.Channel()
	if err != nil {
		log.Panicf("cannot declare a hub channel")
	}

	rooms := make(map[string]*room, 1)
	rooms["hub"] = newRoom("hub", ch)

	g := &Gatekeeper{
		dialer:     d,
		unregister: make(chan *client),
		bus:        make(chan ClientEvent),
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

	g.rooms["hub"].broadcast(ServerEvent{
		Act:     CLIENTS_COUNTER,
		Payload: encodeOrPanic(g.clientsCounter),
	})
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

	g.rooms["hub"].broadcast(ServerEvent{
		Act:     CLIENTS_COUNTER,
		Payload: encodeOrPanic(g.clientsCounter),
	})
}

/*
route forwards events to the room if the room exists and the publisher is
subscribed to that room.
*/
func (g *Gatekeeper) route(e ClientEvent) {
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

	r.publish(e)
}

/*
Destroy destroys all rooms in a gatekeeper.
*/
func (g *Gatekeeper) Destroy() {
	for _, r := range g.rooms {
		r.destroy()
	}
}
