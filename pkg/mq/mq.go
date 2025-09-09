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
DeclareTopology declares the hub topic exchange and two queues: gate and core.
After declaring the queues, they are bound to the exchange, which serves as the
event bus for all events in the system.

The gate queue is consumed by the core server, and the gatekeeper will publish
incomming client events to it.  The core queue is consumed by the gatekeeper,
and the core server will publish server events to it.

Do not call this function from the core server, since it's a gatekeeper's
responsibility to manage the lifecycle of queues and the exchange.
*/
func DeclareTopology(ch *amqp091.Channel) error {
	err := ch.ExchangeDeclare("hub", "topic", false, true, false, false, nil)
	if err != nil {
		return err
	}

	gate, err := ch.QueueDeclare("gate", false, true, false, false, nil)
	if err != nil {
		return err
	}

	core, err := ch.QueueDeclare("core", false, true, false, false, nil)
	if err != nil {
		return err
	}

	err = ch.QueueBind(gate.Name, gate.Name, "hub", false, nil)
	if err != nil {
		return err
	}

	return ch.QueueBind(core.Name, core.Name, "hub", false, nil)
}

/*
Consume consumes events from the queue with the specified name.  It will wait
forever for new events until the program shuts down.  After consuming, the event
will be forwarded to the handle channel.

Panics if the queue cannot be consumed.
*/
func Consume(ch *amqp091.Channel, name string, handle chan<- types.InternalEvent) {
	events, err := ch.Consume(name, "", false, true, false, false, nil)
	if err != nil {
		log.Panicf("cannot consume queue \"%s\": %s", name, err)
		return
	}

	forever := make(<-chan struct{})

	go func() {
		for d := range events {
			var e types.InternalEvent
			if err := json.Unmarshal(d.Body, &e); err != nil {
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
specified channel.  It waits up to 5 seconds for the event to be published;
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
