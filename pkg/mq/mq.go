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
	Connection *amqp091.Connection
}

/*
NewDialer connects to the RabbitMQ using a URL provided as an environment
variable.  Panics if the connection cannot be established.
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
func (d Dialer) OpenChannel() (*amqp091.Channel, error) {
	ch, err := d.Connection.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		return nil, err
	}

	return ch, err
}

/*
DeclareAndBindQueues uses the provided channel to declare two queues, in and
out.  Each queue name is prefixed with the specified name.  After declaring the
queues, they are bound to the "hub" exchange, which serves as the event bus for
all events in the system.

Gatekeeper should not call this function.  Each room is only a publisher and
consumer to the queues, created by the core server.  That way it will be easier
to manage the lifecycle of the queues.
*/
func DeclareAndBindQueues(ch *amqp091.Channel, name string) error {
	in, err := ch.QueueDeclare(name+".in", false, true, true, false, nil)
	if err != nil {
		return err
	}

	out, err := ch.QueueDeclare(name+".out", false, true, true, false, nil)
	if err != nil {
		return err
	}

	err = ch.QueueBind(in.Name, in.Name, "hub", false, nil)
	if err != nil {
		return err
	}

	err = ch.QueueBind(out.Name, out.Name, "hub", false, nil)
	return err
}

/*
Consume consumes events from the queue with the specified name.  It will wait
for new events until the caller sends a signal on a stop channel.  It is the
caller's responsibility to stop the consuming process.  After consuming, the
event will be forwarded to the callback function.

Panics if the queue cannot be consumed.
*/
func Consume(ch *amqp091.Channel, name string, stop <-chan struct{}, callback func([]byte)) {
	events, err := ch.Consume(name, "", false, true, false, false, nil)
	if err != nil {
		log.Panicf("cannot consume queue \"%s\": %s", name, err)
		return
	}

	go func() {
		for d := range events {
			if err != nil {
				log.Printf("cannot unmarshal queue event: %s", err)
				return
			}
			callback(d.Body)
		}
	}()

	<-stop
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
