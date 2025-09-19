package main

import (
	"log"
	"net/http"
	"os"

	"github.com/treepeck/gatekeeper/internal/ws"
	"github.com/treepeck/gatekeeper/pkg/mq"

	"github.com/rabbitmq/amqp091-go"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	log.Print("Connecting to RabbitMQ.")
	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	// Open the AMQP channel.
	ch, err := conn.Channel()
	if err != nil {
		log.Panic(err)
	}
	defer ch.Close()

	// Put the channel into a confirm mode.
	err = ch.Confirm(false)
	if err != nil {
		log.Panic(err)
	}

	// Declare the MQ topology.  See the doc/arch.png file.
	err = mq.DeclareTopology(ch)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Successfully connected to RabbitMQ.")

	log.Print("Starting server.")
	s := ws.NewServer(ch)

	// Run the goroutines which will run untill the program exits.
	go s.Run()
	go mq.Consume(s.Channel, "core", s.EventBus)

	// Handle incomming requests.
	http.HandleFunc("GET /ws", func(rw http.ResponseWriter, r *http.Request) {
		h := ws.Handshake{
			Request:         r,
			ResponseWriter:  rw,
			ResponseChannel: make(chan struct{}),
		}

		s.Register <- h
		// Wait for the response from the handler.
		<-h.ResponseChannel
	})

	log.Panic(http.ListenAndServe(":3503", nil))
}
