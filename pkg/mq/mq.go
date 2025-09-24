/*
Package mq containts helper functions to make the work with RabbitMQ easier.
*/
package mq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"github.com/treepeck/gatekeeper/pkg/types"
)

/*
Consume consumes events from the queue with the specified name.  It will wait
forever for new events until the program shuts down.  After consuming, the event
will be forwarded to the handle channel.

Panics if the queue cannot be consumed.
*/
func Consume(ch *amqp091.Channel, name string, handle chan<- types.MetaEvent) {
	events, err := ch.Consume(name, "", false, true, false, false, nil)
	if err != nil {
		log.Panicf("cannot consume queue \"%s\": %s", name, err)
		return
	}

	forever := make(<-chan struct{})

	go func() {
		var e types.MetaEvent
		for d := range events {
			if err := json.Unmarshal(d.Body, &e); err != nil {
				log.Print(string(d.Body))
				log.Printf("cannot unmarshal queue event: %s", err)
				d.Nack(false, false)
				return
			}

			handle <- e

			// Acknowledge the recieved event.
			d.Ack(false)
		}
	}()

	// forever will always hang.
	<-forever
}

/*
Publish publishes an event to the queue with the specified name via the
specified channel.  Waits up to 5 seconds for the event to be published;
otherwise, an error is logged.
*/
func Publish(ch *amqp091.Channel, name string, raw []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ch.PublishWithContext(
		ctx,
		"hub",
		name,
		false,
		false,
		amqp091.Publishing{
			Body:        raw,
			ContentType: "application/json",
		},
	)
	if err != nil {
		log.Printf("cannot publish a message: %s", err)
	}
}
