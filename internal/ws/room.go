package ws

import (
	"encoding/json"
	"log"

	"github.com/treepeck/gatekeeper/pkg/types"
)

/*
room stores the collection of subscribers and broadcasts events among them.
*/
type room struct {
	subs map[string]*client
}

func newRoom() *room {
	return &room{
		subs: make(map[string]*client, 0),
	}
}

func (r *room) subscribe(c *client) {
	r.subs[c.id] = c
}

func (r *room) unsubscribe(id string) {
	delete(r.subs, id)
}

/*
broadcast encodes the specified event and sends it to all subscribed clients.
*/
func (r *room) broadcast(e types.Event) {
	raw, err := json.Marshal(e)
	if err != nil {
		log.Printf("cannot encode external event: %s", err)
		return
	}

	for _, c := range r.subs {
		c.send <- raw
	}
}
