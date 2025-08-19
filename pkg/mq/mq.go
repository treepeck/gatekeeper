/*
Package mq manages connection with RabbitMQ and provides functions for spawning
new channels, exchanges, and queues.
*/
package mq

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/BelikovArtem/gatekeeper/pkg/event"
	"github.com/rabbitmq/amqp091-go"
)

/*
Dialer wraps a single AMQP connection to RabbitMQ.  Only a single connection is
used to save resources and be able to handle more WebSocket clients.
*/
type Dialer struct {
	Connection *amqp091.Connection
}

/*
NewDialer connects to the RabbitMQ using a URL provided as an environment
variable.
*/
func NewDialer() Dialer {
	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Panicf("cannot connect to RabbitMQ: %s", err)
	}

	return Dialer{
		Connection: conn,
	}
}

/*
OpenChannel opens a unique channel and puts it into a confirm mode, which allow
waiting for ACK or NACK from the server.
*/
func OpenChannel(conn *amqp091.Connection) (*amqp091.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("cannot open a RabbitMQ channel: %s", err)
		return nil, err
	}

	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		log.Printf("cannot put channel into confirm mode: %s", err)
		return nil, err
	}

	return ch, err
}

/*
DeclareTopology declares the "HUB" topic exchange.

Call this function ONLY on the core server as close to the program start as
possible.
*/
func DeclareExchange(ch *amqp091.Channel) error {
	err := ch.ExchangeDeclare("hub", "topic", false, true, false, false, nil)
	if err != nil {
		log.Printf("cannot declare an exchange: %s", err)
		return err
	}
	return err
}

/*
DeleteExchange deletes the "HUB" topic exchange.

Call this function on the core server exit to cleanup the created exchange and
free resources.
*/
func DeleteExchange(ch *amqp091.Channel) error {
	return ch.ExchangeDelete("hub", false, false)
}

/*
DeclareAndBindQueues uses the provided channel to declare two queues, in and
out.  Each queue name is prefixed with the exchange’s name.  After declaring the
queues, they are bound to the gatekeeper exchange, which serves as the event bus
for all messages in the system.

Each room must call this function to declare unique queues and use the roomId as
a name.
*/
func DeclareAndBindQueues(ch *amqp091.Channel, name string) error {
	iQ, err := ch.QueueDeclare(name+".in", false, true, true, false, nil)
	if err != nil {
		log.Printf("cannot create in queue: %s", err)
		return err
	}

	oQ, err := ch.QueueDeclare(name+".out", false, true, true, false, nil)
	if err != nil {
		log.Printf("cannot create out queue: %s", err)
		return err
	}

	err = ch.QueueBind(iQ.Name, iQ.Name, "hub", false, nil)
	if err != nil {
		log.Printf("cannot bind \"%s\" queue to exchange: %s", iQ.Name, err)
		return err
	}

	err = ch.QueueBind(oQ.Name, oQ.Name, "hub", false, nil)
	if err != nil {
		log.Printf("cannot bind \"%s\" queue to exchange: %s", oQ.Name, err)
	}
	return err
}

/*
ConsumeQueue consumes a queue with the specified name.  It wil wait for a new
messages until the caller will send a signal on a stop channel.  It's a caller's
resonsibility to stop the consuming process.  After consuming, the event will be
forwarded into the callback function.

Panics if the queue cannot be consumed.
*/
func Consume(ch *amqp091.Channel, name string, stop <-chan struct{},
	callback func(event.ServerEvent)) {

	events, err := ch.Consume(name, "", false, true, false, false, nil)
	if err != nil {
		log.Panicf("cannot consume queue \"%s\": %s", name, err)
		return
	}

	go func() {
		for d := range events {
			var e event.ServerEvent
			err := json.Unmarshal(d.Body, &e)
			if err != nil {
				log.Printf("cannot unmarshal queue event: %s", err)
				return
			}
			callback(e)
		}
	}()

	<-stop
}

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

/*
Release deletes the gatekeeper exchange and
*/
func (d Dialer) Release() {
	d.Connection.Close()
}
