/*
Package mq manages connection with RabbitMQ and provides functions for spawning
new channels, exchanges, and queues.
*/
package mq

import (
	"log"
	"os"

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
variable.  After establishing a connection, a new channel is opened.
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
DeclareTopology declares an topic exchange and two queues.
*/
func DeclareTopology(ch *amqp091.Channel) error {
	err := ch.ExchangeDeclare("hub", "topic", false, true, false, false, nil)
	if err != nil {
		log.Printf("cannot declare an exchange: %s", err)
		return err
	}

	err = DeclareAndBindQueues(ch, "hub")
	if err != nil {
		log.Printf("cannot declare and bind queues: %s", err)
	}
	return err
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
ConsumeQueue consumes a queue with the specified name.
*/
func ConsumeQueue(ch *amqp091.Channel, name string) (<-chan amqp091.Delivery, error) {
	return ch.Consume(name, "", false, true, false, false, nil)
}

/*
Release deletes the gatekeeper exchange and
*/
func (d Dialer) Release() {
	d.Connection.Close()
}
