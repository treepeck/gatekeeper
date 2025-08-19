package main

import (
	"log"
	"net/http"
	"os"

	"github.com/BelikovArtem/gatekeeper/internal/ws"
	"github.com/BelikovArtem/gatekeeper/pkg/env"
	"github.com/BelikovArtem/gatekeeper/pkg/mq"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	env.Load(".env")

	d := mq.NewDialer()
	defer d.Release()

	g := ws.NewGatekeeper(d)
	defer g.Destroy()

	http.HandleFunc("GET /ws", func(rw http.ResponseWriter, r *http.Request) {
		ws.HandleHandshake(rw, r, g)
	})

	http.ListenAndServe(os.Getenv("ADDR"), nil)
}
