package types

/*
EventAction represents a domain of possible event actions.
*/
type EventAction int

const (
	// Events that can be sent by both Gatekeeper and core server.
	ActionPing EventAction = iota
	ActionGameInfo
	ActionRedirect
	ActionCompletedMove
	ActionClientsCounter
	// Events that can be sent only by the clients.
	ActionPong
	ActionMakeMove
	ActionEnterMatchmaking
	// Events that can be sent by both Gatekeeper and clients.
	ActionChat
	// Events that can be sent only by the core server.
	ActionAddRoom
	ActionRemoveRoom
)

type Event struct {
	Action  EventAction `json:"a"`
	Payload any         `json:"p"`
}

type EnterMatchmaking struct {
	ClientId    string `json:"cid"`
	TimeControl int    `json:"tc"`
	TimeBonus   int    `json:"tb"`
}

type AddRoom struct {
	RoomId  string `json:"rid"`
	WhiteId string `json:"wid"`
	BlackId string `json:"bid"`
}

type MakeMove struct {
	ClientId string `json:"cid"`
	RoomId   string `json:"rid"`
	Move     int    `json:"m"`
}

type Chat struct {
	ClientId string `json:"cid"`
	RoomId   string `json:"rid"`
	Message  string `json:"msg"`
}

type CompletedMove struct {
	// Legal moves for the next turn.
	LegalMoves []int  `json:"lm"`
	RoomId     string `json:"rid"`
	// Completed move in Standard Algebraic Notation.
	San string `json:"san"`
	// Board state in Forsyth-Edwards Notation.
	Fen string `json:"fen"`
	// Remaining seconds on the player's clock after completing the move.
	TimeLeft int `json:"tl"`
}

/*
GameInfo represents information the clients will recieve after connecting
to the room.
*/
type GameInfo struct {
	RoomId      string `json:"rid"`
	WhiteId     string `json:"wid"`
	BlackId     string `json:"bid"`
	TimeControl int    `json:"tc"`
	TimeBonus   int    `json:"tb"`
}
