package ws

import (
	"context"
	"encoding/json"
	"log"
	"time"

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

	if r.id == "hub" {
		mq.DeclareTopology(r.channel)
	} else {
		mq.DeclareAndBindQueues(r.channel, r.id)
	}

	go r.consume()

	return r
}

/*
subscribe subscribes the specified client to the room.
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
publish publishes events to the room 'out' queue.
*/
func (r *room) publish(e ClientEvent) {
	body, err := json.Marshal(e)
	if err != nil {
		log.Panicf("cannot marshal a body")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = r.channel.PublishWithContext(
		ctx,
		"hub",
		r.id+"out",
		false,
		false,
		amqp091.Publishing{
			Body:        body,
			ContentType: "application/json",
		},
	)
	if err != nil {
		log.Panicf("cannot publish a message: %s", err)
	}
}

/*
consume consumes events from the room 'in' queue until recieves a singal from
stopConsuming channel.  Each recieves event is broadcasted among all subscribed
clients.
*/
func (r *room) consume() {
	events, err := mq.ConsumeQueue(r.channel, r.id+"in")
	if err != nil {
		log.Panicf("cannot consume queue \"%s\": %s", r.id+"in", err)
		return
	}

	go func() {
		for d := range events {
			var e ServerEvent
			err := json.Unmarshal(d.Body, &e)
			if err != nil {
				log.Panicf("cannot unmarshal queue event: %s", err)
			}
			r.broadcast(e)
		}
	}()

	<-r.stopConsuming
}

/*
broadcast broadcasts the event among all subscribed clients.
*/
func (r *room) broadcast(e ServerEvent) {
	for _, c := range r.subs {
		c.send <- e
	}
}

/*
destroy stops the [consume] goroutine and closes the room channel to prevent
memory leaks.
*/
func (r *room) destroy() {
	if r.id == "hub" {
		err := r.channel.ExchangeDelete(r.id, false, false)
		if err != nil {
			log.Printf("cannot delete a exchange: %s", err)
		}
	}

	r.stopConsuming <- struct{}{}

	for _, c := range r.subs {
		close(c.send)
	}

	r.channel.Close()
}
