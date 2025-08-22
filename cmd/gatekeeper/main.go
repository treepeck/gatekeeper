package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rabbitmq/amqp091-go"

	"github.com/BelikovArtem/gatekeeper/internal/ws"
	"github.com/BelikovArtem/gatekeeper/pkg/env"
	"github.com/BelikovArtem/gatekeeper/pkg/mq"
)

func main() {
	// Set up logger.
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	// Load environment variables.
	if err := env.Load(".env"); err != nil {
		log.Fatal(err)
	}

	// Connect to RabbitMQ.
	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// Open an AMQP channel.
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Fatal(err)
	}

	// Put the channel into a confirm mode.
	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Fatal(err)
	}

	// Declare the MQ topology.  See the doc/arch.png file.
	err = mq.DeclareTopology(ch)
	if err != nil {
		log.Fatal(err)
	}

	s := ws.NewServer(ch)

	// Run the goroutines which will run untill the program exits.
	go s.Run()
	go mq.Consume(ch, "core", s.Bus)

	// Handle incomming requests.
	http.HandleFunc("GET /ws", func(rw http.ResponseWriter, r *http.Request) {
		ws.HandleHandshake(rw, r, s)
	})
	http.ListenAndServe(os.Getenv("ADDR"), nil)
}
