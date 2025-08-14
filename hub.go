package main

import (
	"crypto/rand"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Hub struct {
	topics         map[string]*topic
	register       chan *websocket.Conn
	unregister     chan *client
	create         chan *topic
	remove         chan string
	route          chan event
	clientsCounter int
}

func NewHub() *Hub {
	topics := make(map[string]*topic)
	topics["hub"] = newTopic("hub")

	h := &Hub{
		topics:     topics,
		register:   make(chan *websocket.Conn),
		unregister: make(chan *client),
		create:     make(chan *topic),
		remove:     make(chan string),
		route:      make(chan event),
	}

	go h.bus()

	return h
}

func (h *Hub) bus() {
	for {
		select {
		case conn := <-h.register:
			h.handleRegister(conn)

		case c := <-h.unregister:
			h.handleUnregister(c)

		case r := <-h.create:
			h.handleCreate(r)

		case roomId := <-h.remove:
			h.handleRemove(roomId)

		case e := <-h.route:
			h.routeEvent(e)
		}
	}
}

func (h *Hub) handleRegister(conn *websocket.Conn) {
	h.clientsCounter++

	// TODO: parse room id from the request parameter.
	c := newClient(rand.Text(), "hub", h, conn)
	h.topics["hub"].subscribe(c)

	p, _ := json.Marshal(h.clientsCounter)
	h.broadcast(event{TopicId: "hub", Act: counter, Payload: p})
}

func (h *Hub) handleUnregister(c *client) {
	t := h.topics[c.topicId]
	if t != nil && t.isSub(c) {
		t.unsubscribe(c)

		h.clientsCounter--

		p, _ := json.Marshal(h.clientsCounter)
		h.broadcast(event{TopicId: "hub", Act: counter, Payload: p})
	}
}

func (h *Hub) handleCreate(t *topic) {

}

func (h *Hub) handleRemove(roomId string) {

}

func (h *Hub) routeEvent(e event) {

}

func (h *Hub) broadcast(e event) {
	if t, exists := h.topics[e.TopicId]; exists {
		for sub := range t.subs {
			sub.send <- e
		}
	}
}
