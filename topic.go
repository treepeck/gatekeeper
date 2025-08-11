package pubsub

type Topic struct {
	// Topic id.
	id string
	// Subscribed actors.
	subs map[*Actor]struct{}
	// handlers will handle the incomming events.
	handlers map[Action]Handler
	// Recieve channel will recieve the events from the subscribed actors.
	Recieve chan Event
	// Close will stop the RouteEvents routine.
	Destroy chan bool
}

func NewTopic(id string, handlers map[Action]Handler) *Topic {
	return &Topic{
		id:       id,
		subs:     make(map[*Actor]struct{}),
		handlers: make(map[Action]Handler),
		Recieve:  make(chan Event),
		Destroy:  make(chan bool),
	}
}

func (t *Topic) RouteEvents() {
	for {
		select {
		case e := <-t.Recieve:
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

func (t *Topic) publish(data []byte) {
	for a := range t.subs {
		a.Recieve <- data
	}
}
