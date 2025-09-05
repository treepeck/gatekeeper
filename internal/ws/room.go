package ws

import (
	"github.com/treepeck/gatekeeper/pkg/event"
)

/*
room describes the behavoiur of the arbitrary room.  The are two room types:
hub and game.  The hub room will handle events differently, that's why the
interface is used (to be able to store different structures in a single map).
*/
type room interface {
	subscribe(id string, c *client)
	unsubscribe(id string)
	broadcast(e event.ExternalEvent)
}

type hubRoom struct {
	subs map[string]*client
}

func newHubRoom() *hubRoom {
	return &hubRoom{
		subs: make(map[string]*client, 0),
	}
}

func (r *hubRoom) subscribe(id string, c *client) {
	r.subs[id] = c
}

func (r *hubRoom) unsubscribe(id string) {
	delete(r.subs, id)
}

func (r *hubRoom) broadcast(e event.ExternalEvent) {
	for _, c := range r.subs {
		c.send <- e
	}
}

/*
gameRoom represents a single active game.
*/
type gameRoom struct {
	subs map[string]*client
}

func newGameRoom() *gameRoom {
	return &gameRoom{
		subs: make(map[string]*client),
	}
}

func (r *gameRoom) subscribe(id string, c *client) {
	r.subs[id] = c
}

func (r *gameRoom) unsubscribe(id string) {
	delete(r.subs, id)
}

func (r *gameRoom) broadcast(e event.ExternalEvent) {
	for _, c := range r.subs {
		c.send <- e
	}
}
