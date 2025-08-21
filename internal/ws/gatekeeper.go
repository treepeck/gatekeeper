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
	subs           map[string]map[*client]struct{}
}

/*
NewCore opens an AMQP channel, declares the exchange and creates a new Gatekeeper
instance.
*/
func NewGatekeeper(d mq.Dialer) *Gatekeeper {
	subs := make(map[string]map[*client]struct{}, 1)
	subs["hub"] = make(map[*client]struct{}, 0)

	g := &Gatekeeper{
		dialer:     d,
		unregister: make(chan *client),
		bus:        make(chan event.ClientEvent),
		Register:   make(chan *websocket.Conn),
		subs:       subs,
	}

	go g.routeEvents()

	go d.Consume("core", g.consume)

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
			g.publish(e)
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

	c.roomId = "hub"

	g.subs["hub"][c] = struct{}{}

	raw := event.EncodeOrPanic(event.ServerEvent{
		Action:  event.CLIENTS_COUNTER,
		Payload: event.EncodeOrPanic(g.clientsCounter),
	})
	g.notify("hub", raw)
}

/*
handleUnregister unregisters the client and broadcasts the clientCounter among
all subscribed to the "hub" room clients.
*/
func (g *Gatekeeper) handleUnregister(c *client) {
	g.clientsCounter--

	if _, exists := g.subs[c.roomId]; !exists {
		log.Printf("client \"%s\" is subscribed to \"%s\" which doesn't exist",
			c.id, c.roomId)
		return
	}

	delete(g.subs[c.roomId], c)

	raw := event.EncodeOrPanic(event.ServerEvent{
		Action:  event.CLIENTS_COUNTER,
		Payload: event.EncodeOrPanic(g.clientsCounter),
	})
	g.notify("hub", raw)
}

/*
publish publishes client event to the room's out queue if the room exists and the
publisher is subscribed to that room.
*/
func (g *Gatekeeper) publish(e event.ClientEvent) {
	if _, exists := g.subs[e.RoomId]; !exists {
		log.Printf("client \"%s\" tries to publish to \"%s\" which doesn't exist",
			e.ClientId, e.RoomId)
		// TODO: disconnect the client.
		return
	}

	raw, err := json.Marshal(e)
	if err != nil {
		log.Printf("cannot encode event from \"%s\"", e.ClientId)
		return
	}
	g.dialer.Publish("gate", raw)
}

/*
consume notifies the subscribed clients about the consumed server event.
Gatekeeper always ACK's the consuming events.
*/
func (g *Gatekeeper) consume(raw []byte) error {
	// TODO: route the event to the corresponding room.
	g.notify("hub", raw)

	return nil
}

/*
notify sends the specified raw message to all clients, subscribed to the roomId.
*/
func (g *Gatekeeper) notify(roomId string, raw []byte) {
	for sub := range g.subs[roomId] {
		sub.send <- raw
	}
}
