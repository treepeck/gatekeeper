package main

import "encoding/json"

type eventAct int

const (
	counter eventAct = iota
)

type event struct {
	Payload  json.RawMessage `json:"p"`
	ClientId string          `json:"-"`
	TopicId  string          `json:"-"`
	Act      eventAct        `json:"a"`
}
