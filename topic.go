package pubsub

import "log"

type topic struct {
	subs map[string]*Actor
}

func newTopic() *topic {
	return &topic{
		subs: make(map[string]*Actor, 0),
	}
}

func (t *topic) subscribe(a *Actor) {
	log.Printf("actor \"%s\" subscribed to topic \"%s\"", a.id, a.topId)
	t.subs[a.id] = a
}

func (t *topic) unsubscribe(a *Actor) {
	log.Printf("actor \"%s\" ubsubscribed from topic \"%s\"", a.id, a.topId)
	delete(t.subs, a.id)
}

func (t *topic) publish(e Event) {
	for _, a := range t.subs {
		a.Recieve <- e.Data
	}
}
