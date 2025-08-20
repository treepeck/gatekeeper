package ws

import (
	"log"

	"github.com/BelikovArtem/gatekeeper/pkg/mq"
	"github.com/rabbitmq/amqp091-go"
)

/*
room forwards incomming events into the "out" queue and broadcasts events recieved
from the "in" queue back to the subscribed clients.
*/
type room struct {
	id            string
	channel       *amqp091.Channel
	subs          map[string]*client
	stopConsuming chan struct{}
}

func newRoom(id string, ch *amqp091.Channel) *room {
	r := &room{
		id:            id,
		channel:       ch,
		subs:          make(map[string]*client),
		stopConsuming: make(chan struct{}),
	}

	mq.DeclareAndBindQueues(r.channel, r.id)

	go mq.Consume(r.channel, r.id+".in", r.stopConsuming, r.broadcast)

	return r
}

/*
subscribe subscribes the specified client to the room.  Each subscribed client
will recieve a redirect event with the room id.
*/
func (r *room) subscribe(c *client) {
	if _, exists := r.subs[c.id]; exists {
		log.Printf("client \"%s\" tries to subscribe multiple times", c.id)
		c.cleanup()
		return
	}

	r.subs[c.id] = c
	c.roomId = r.id

	log.Printf("client \"%s\" subscribed to \"%s\"", c.id, r.id)
}

/*
unsubscribe unsubscribed the specified client from the room.
*/
func (r *room) unsubscribe(c *client) {
	if _, exists := r.subs[c.id]; !exists {
		log.Printf("client \"%s\" tries to unsubscribe but isn't subscribed", c.id)
	}

	delete(r.subs, c.id)

	log.Printf("client \"%s\" unsubscribed from \"%s\"", c.id, r.id)
}

/*
broadcast broadcasts the event among all subscribed clients.
*/
func (r *room) broadcast(raw []byte) {
	for _, c := range r.subs {
		c.send <- raw
	}
}

/*
destroy stops the [consume] goroutine and closes the room channel to prevent
memory leaks.
*/
func (r *room) destroy() {
	r.stopConsuming <- struct{}{}

	for _, c := range r.subs {
		close(c.send)
	}

	r.channel.Close()
}
