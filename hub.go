package pubsub

import "log"

type Hub struct {
	Register   chan *Actor
	Unregister chan *Actor
	Bus        chan Event
	topics     map[TopicId]*topic
}

func NewHub() *Hub {
	// Add default topic.
	topics := make(map[TopicId]*topic, 2)
	topics[HUB] = newTopic()

	return &Hub{
		Register:   make(chan *Actor),
		Unregister: make(chan *Actor),
		Bus:        make(chan Event),
		topics:     topics,
	}
}

func (h *Hub) RouteEvents() {
	for {
		select {
		case a := <-h.Register:
			h.registerActor(a)

		case a := <-h.Unregister:
			h.unregisterActor(a)

		case e := <-h.Bus:
			switch e.Action {
			case CREATE:
				h.createTopic(TopicId(e.Data))

			case SUBSCRIBE:
				h.subscribe(e.Pub, TopicId(e.Data))

			case PUBLISH:
				h.getOrPanic(e.TopId).publish(e)
			}
		}
	}
}

func (h *Hub) registerActor(a *Actor) {
	a.bus = h.Bus
	log.Printf("actor \"%s\" registered", a.id)

	h.topics[HUB].subscribe(a)
}

func (h *Hub) unregisterActor(a *Actor) {
	t := h.getOrPanic(a.topId)
	t.unsubscribe(a)

	log.Printf("actor \"%s\" unregistered", a.id)

	if len(t.subs) == 0 && a.topId != HUB {
		h.removeTopic(a.topId)
	}
}

func (h *Hub) createTopic(topId TopicId) {
	t := newTopic()
	h.topics[topId] = t

	log.Printf("topic \"%s\" created", topId)
}

func (h *Hub) removeTopic(topId TopicId) {
	delete(h.topics, topId)
	log.Printf("topic \"%s\" removed", topId)
}

func (h *Hub) subscribe(a *Actor, topId TopicId) {
	h.topics[HUB].unsubscribe(a)

	t := h.getOrPanic(topId)
	a.topId = topId
	t.subscribe(a)

}

func (h *Hub) getOrPanic(topId TopicId) *topic {
	t, ok := h.topics[topId]
	if !ok {
		panic("trying to get a non-existent topic")
	}
	return t
}
