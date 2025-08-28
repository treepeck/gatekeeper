package main

import (
	"log"
	"net/http"
	"os"

	"github.com/rabbitmq/amqp091-go"

	"github.com/treepeck/gatekeeper/internal/ws"
	"github.com/treepeck/gatekeeper/pkg/env"
	"github.com/treepeck/gatekeeper/pkg/mq"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	log.Print("Loading environment variables.")
	if err := env.Load(".env"); err != nil {
		log.Panic(err)
	}
	log.Print("Successfully loaded environment variables.")

	log.Print("Connecting to RabbitMQ.")
	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	// Open an AMQP channel.
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
	go mq.Consume(ch, "core", s.InternalBus)

	// Handle incomming requests.
	http.HandleFunc("GET /ws", func(rw http.ResponseWriter, r *http.Request) {
		ws.HandleHandshake(rw, r, s)
	})
	http.ListenAndServe(os.Getenv("ADDR"), nil)
}
