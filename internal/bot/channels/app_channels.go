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
	ShutdownChannel          chan struct{}
	SenderChannel            chan Message
	NotifyBroadCasterChannel chan string
)

func InitChannels() {
	ShutdownChannel = make(chan struct{}, 1)
	SenderChannel = make(chan Message, 1000)
	NotifyBroadCasterChannel = make(chan string, 1)
}

func CloseChannels() {
	close(ShutdownChannel)
	close(SenderChannel)
	close(NotifyBroadCasterChannel)
}
