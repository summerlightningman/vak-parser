package common

type BotMsgType int

const (
	BotMsgTypeRunParse BotMsgType = iota
	BotMsgTypeSuccess
	BotMsgTypeParseHasRanAt
)

type SuccessPayload struct {
	Url string
	Page int
}

type BotMsg struct {
	Type BotMsgType
	SuccessPayload SuccessPayload
	Text string
}
