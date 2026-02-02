package channels

import (
	"context"

	"github.com/mymmrac/telego"
)

type Message struct {
	Msg    *telego.SendMessageParams
	UpdCtx context.Context
}

var (
	ShutdownChannel chan struct{}
	SenderChannel   chan Message
)

func InitChannels() {
	ShutdownChannel = make(chan struct{}, 1)
	SenderChannel = make(chan Message, 1000)
}

func CloseChannels() {
	close(ShutdownChannel)
	close(SenderChannel)
}
