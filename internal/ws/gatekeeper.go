package ws

import (
	"crypto/rand"
	"log"
	"net/http"

	"github.com/BelikovArtem/gatekeeper/internal/mq"
	"github.com/gorilla/websocket"
)

// upgrader is used to establish a WebSocket connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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

func (g *Gatekeeper) HandleNewConnection(rw http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		return
	}

	g.Register <- conn
}

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

func (g *Gatekeeper) handleRegister(conn *websocket.Conn) {
	g.clientsCounter++

	c := newClient(rand.Text(), g, conn)
	g.rooms["hub"].subscribe(c)

	g.rooms["hub"].broadcast(ServerEvent{
		Act:     CLIENTS_COUNTER,
		Payload: encodeOrPanic(g.clientsCounter),
	})
}

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

func (g *Gatekeeper) route(e ClientEvent) {
	r, exists := g.rooms[e.RoomId]
	if !exists {
		log.Printf("client \"%s\" sends a message to \"%s\" which doesn't exist",
			e.ClientId, e.RoomId)
		return
	}

	r.publish(e)
}

func (g *Gatekeeper) Destroy() {
	for _, r := range g.rooms {
		r.destroy()
	}
}
