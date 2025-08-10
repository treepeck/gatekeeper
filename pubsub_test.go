package pubsub_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/BelikovArtem/pubsub"
)

func TestActorLifecycle(t *testing.T) {
	h := pubsub.NewHub()
	go h.RouteEvents()

	a := pubsub.NewActor("test_actor")

	h.Register <- a
	time.Sleep(time.Second)

	h.Unregister <- a
	time.Sleep(time.Second)
}

func TestTopicLifecycle(t *testing.T) {
	h := pubsub.NewHub()
	go h.RouteEvents()

	a := pubsub.NewActor("test_actor")

	h.Register <- a
	time.Sleep(time.Second)

	topId := pubsub.TopicId(rand.Text())
	e := pubsub.Event{
		Data:   []byte(topId),
		TopId:  pubsub.HUB,
		Pub:    a,
		Action: pubsub.CREATE,
	}

	h.Bus <- e
	time.Sleep(time.Second)

	e = pubsub.Event{
		Data:   []byte(topId),
		TopId:  pubsub.HUB,
		Pub:    a,
		Action: pubsub.SUBSCRIBE,
	}

	h.Bus <- e
	time.Sleep(time.Second)

	h.Unregister <- a
	time.Sleep(time.Second)
}
