package channels

import "github.com/mymmrac/telego"

var (
	ShutdownChannel chan struct{}
	SenderChannel   chan *telego.SendMessageParams
)

func InitChannels() {
	ShutdownChannel = make(chan struct{}, 1)
	SenderChannel = make(chan *telego.SendMessageParams, 1000)
}
