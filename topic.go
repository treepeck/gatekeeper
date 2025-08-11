package pubsub

import "log"

type Topic struct {
	// Topic Id.
	Id string
	// Subscribed actors.
	subs map[*Actor]struct{}
	// handlers will handle the incomming events.
	handlers map[Action]Handler
	// Register a new subscriber.
	Register chan *Actor
	// Unregister an existing subscriber.
	Unregister chan *Actor
	// Publish channel will recieve the events from the subscribed actors.
	Publish chan Event
	// Close will stop the RouteEvents routine.
	Destroy chan bool
}

func NewTopic(Id string, handlers map[Action]Handler) *Topic {
	return &Topic{
		Id:         Id,
		subs:       make(map[*Actor]struct{}),
		handlers:   make(map[Action]Handler),
		Register:   make(chan *Actor),
		Unregister: make(chan *Actor),
		Publish:    make(chan Event),
		Destroy:    make(chan bool),
	}
}

func (t *Topic) RouteEvents() {
	for {
		select {
		case a := <-t.Register:
			t.register(a)

		case a := <-t.Unregister:
			t.unregister(a)

		case e := <-t.Publish:
			if h, exists := t.handlers[e.Action]; exists {
				if data, err := h(e); err == nil {
					t.publish(data)
				}
			}

		case <-t.Destroy:
			return
		}
	}
}

// register registers a new subscriber.
func (t *Topic) register(a *Actor) {
	t.subs[a] = struct{}{}
	log.Printf("actor \"%s\" registered in topic \"%s\"", a.Id, t.Id)
}

// unregister unregisters an existing subsciber.
func (t *Topic) unregister(a *Actor) {
	if _, exists := t.subs[a]; exists {
		t.subs[a] = struct{}{}
		log.Printf("actor \"%s\" ubregistered from topic \"%s\"", a.Id, t.Id)
	}
}

func (t *Topic) publish(data []byte) {
	for a := range t.subs {
		a.Recieve <- data
	}
}
