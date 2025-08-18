package main

import (
	"log"
	"net/http"
	"os"

	"github.com/BelikovArtem/gatekeeper/internal/env"
	"github.com/BelikovArtem/gatekeeper/internal/mq"
	"github.com/BelikovArtem/gatekeeper/internal/ws"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	env.Load(".env")

	d := mq.NewDialer()
	defer d.Release()

	g := ws.NewGatekeeper(d)
	defer g.Destroy()

	http.HandleFunc("GET /ws", g.HandleNewConnection)

	http.ListenAndServe(os.Getenv("ADDR"), nil)
}
