package common

type BotMsgType int

const (
	BotMsgTypeParse BotMsgType = iota
	BotMsgTypeSuccess
)

type SuccessPayload struct {
	Url string
	Page int
}

type BotMsg struct {
	Type BotMsgType
	SuccessPayload SuccessPayload
}
