package ws

import "log"

type room struct {
	timeControl int
	timeBonus   int
	subs        map[string]struct{}
}

func newRoom(timeControl int, timeBonus int) *room {
	return &room{
		timeControl: timeControl,
		timeBonus:   timeBonus,
		subs:        make(map[string]struct{}, 2),
	}
}

func (r *room) subscribe(id string) {
	log.Printf("client \"%s\" subscribed to the room", id)
}

func (r *room) unsubscribe(id string) {
	log.Printf("client \"%s\" unsubscribed from the room", id)
}

func (r *room) broadcast(raw []byte) {

}
