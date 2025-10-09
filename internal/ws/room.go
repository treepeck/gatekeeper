package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/treepeck/gatekeeper/pkg/types"
)

/*
room stores the collection of subscribers and broadcasts events among them.

There are two types of rooms:
  - Hub room simply fan-outs the incomming server events to the connected clients.
  - Game room does the same but additionaly caches the last known game info after
    each completed move to quickly send it to the connected clients without the
    need to fetch it from the core server.
*/
type room interface {
	subscribe(c *client)
	unsubscribe(id string)
	broadcast(e types.ExternalEvent)
	// destroy closes the connection of each subscribed client.
	destroy()
	getSubscriber(clientId string) *client
}

type hubRoom struct {
	subs map[string]*client
}

func newHubRoom() *hubRoom {
	return &hubRoom{subs: make(map[string]*client)}
}

func (r *hubRoom) subscribe(c *client) {
	r.subs[c.id] = c
}

func (r *hubRoom) unsubscribe(id string) {
	delete(r.subs, id)
}

/*
broadcast encodes the specified event and sends it to all subscribed clients.
*/
func (r *hubRoom) broadcast(e types.ExternalEvent) {
	raw, err := json.Marshal(e)
	if err != nil {
		log.Printf("cannot encode external event: %s", err)
		return
	}

	for _, c := range r.subs {
		c.send <- raw
	}
}

func (r *hubRoom) destroy() {
	for _, c := range r.subs {
		c.conn.Close()
	}
}

func (r *hubRoom) getSubscriber(clientId string) *client {
	return r.subs[clientId]
}

type gameRoom struct {
	subs     map[string]*client
	info     types.GameInfo
	lastInfo time.Time
}

func newGameRoom(whiteId, blackId string) *gameRoom {
	i := types.GameInfo{
		CompletedMoves: make([]types.CompletedMove, 0),
		WhiteId:        whiteId,
		BlackId:        blackId,
	}

	return &gameRoom{
		subs:     make(map[string]*client, 0),
		info:     i,
		lastInfo: time.Now(),
	}
}

/*
subscribe sends the cached game state to the connected client.
*/
func (r *gameRoom) subscribe(c *client) {
	r.subs[c.id] = c

	p, err := json.Marshal(r.info)
	if err != nil {
		log.Printf("cannot encode game info: %s", err)
	}

	raw, err := json.Marshal(types.ExternalEvent{
		Action:  types.ActionGameInfo,
		Payload: p,
	})
	if err != nil {
		log.Printf("cannot encode external event: %s", err)
		return
	}

	c.send <- raw
}

func (r *gameRoom) unsubscribe(id string) {
	delete(r.subs, id)
}

func (r *gameRoom) broadcast(e types.ExternalEvent) {
	// Cache game info after each move.
	if e.Action == types.ActionCompletedMove {
		var p types.CompletedMove
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			log.Printf("cannot decode completed move: %s", err)
			return
		}
		r.info.CompletedMoves = append(r.info.CompletedMoves, p)
	}

	raw, err := json.Marshal(e)
	if err != nil {
		log.Printf("cannot encode external event: %s", err)
		return
	}

	for _, c := range r.subs {
		c.send <- raw
	}
}

func (r *gameRoom) destroy() {
	for _, c := range r.subs {
		c.conn.Close()
	}
}

func (r *gameRoom) getSubscriber(clientId string) *client {
	return r.subs[clientId]
}
