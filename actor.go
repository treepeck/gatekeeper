package pubsub

type Actor struct {
	id      string
	topId   TopicId
	bus     chan<- Event
	Recieve chan []byte
}

func NewActor(id string) *Actor {
	return &Actor{
		id: id,
		// Actor is subscribed to the HUB topic by default.
		topId: HUB,
		// Hub's event bus to be able to publish events. nil by default.
		bus: nil,
		// Recieve channel will fetch the events from the topic to which
		// the actor is subscribed.
		Recieve: make(chan []byte, 192),
	}
}

func (a *Actor) Publish(e Event) {
	a.bus <- e
}
