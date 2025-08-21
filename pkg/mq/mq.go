package mq

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

/*
Dialer wraps a single AMQP connection to RabbitMQ.  Only a single connection is
used to save resources and be able to handle more WebSocket clients.
*/
type Dialer struct {
	connection *amqp091.Connection
	channel    *amqp091.Channel
}

/*
NewDialer connects to the RabbitMQ using a URL provided as an environment
variable.  Also opens a new channel after the connection is established.  Panics
if the connection cannot be established or the channel cannot be opened.
*/
func NewDialer() Dialer {
	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Panicf("cannot connect to RabbitMQ: %s", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Panicf("cannot open a channel: %s", err)
	}

	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Panicf("cannot put the channel into a confirm mode: %s", err)
	}

	return Dialer{
		connection: conn,
		channel:    ch,
	}
}

/*
DeclareTopology declares the hub topic exchange, and two queues: gate and core.
After declaring the queues, they are bound to the exchange, which serves as the
event bus for all events in the system.

The gate queue is consumed by the core server, and the gatekeeper will publish
incomming client events to it.  The core queue is consumed by the gatekeeper,
and the core server will publish server events to it.

Do not call this function from the core server, since it's a gatekeeper's
responsibility to manage the lifecycle of queues and the exchange.
*/
func (d Dialer) DeclareTopology() error {
	err := d.channel.ExchangeDeclare("hub", "topic", false, true, false, false, nil)
	if err != nil {
		return err
	}

	gate, err := d.channel.QueueDeclare("gate", false, true, false, false, nil)
	if err != nil {
		return err
	}

	core, err := d.channel.QueueDeclare("core", false, true, false, false, nil)
	if err != nil {
		return err
	}

	err = d.channel.QueueBind(gate.Name, gate.Name, "hub", false, nil)
	if err != nil {
		return err
	}

	err = d.channel.QueueBind(core.Name, core.Name, "hub", false, nil)
	return err
}

/*
Consume consumes events from the queue with the specified name.  It will wait
forever for new events until the Gatekeeper programs exists.  After consuming,
the event will be forwarded to the callback function.

Panics if the queue cannot be consumed.
*/
func (d Dialer) Consume(name string, callback func([]byte)) {
	events, err := d.channel.Consume(name, "", false, true, false, false, nil)
	if err != nil {
		log.Panicf("cannot consume queue \"%s\": %s", name, err)
		return
	}

	forever := make(<-chan struct{})

	go func() {
		for d := range events {
			if err != nil {
				log.Printf("cannot unmarshal queue event: %s", err)
				return
			}
			callback(d.Body)
		}
	}()

	// Wait forever.
	<-forever
}

/*
Publish publishes an event to the queue with the specified name via the
specified channel.  It waits up to 5 seconds for the event to be published;
otherwise, an error is logged.
*/
func (d Dialer) Publish(name string, raw []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := d.channel.PublishWithContext(
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
