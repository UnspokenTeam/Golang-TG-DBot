package channels

import (
	"context"
	"time"

	"github.com/mymmrac/telego"
)

type Message struct {
	Msg    *telego.SendMessageParams
	UpdCtx context.Context
}

type FeedbackResult struct {
	Success     bool
	ChatId      int64
	Message     string
	Err         error
	AttemptedAt time.Time
}

type BroadcastTask struct {
	BatchCtx              context.Context
	ChatId                int64
	Text                  string
	BatchId               string
	SenderFeedbackChannel chan FeedbackResult
}

type BroadcastBase struct {
	Text    string
	BatchId string
}

var (
	ShutdownChannel          chan struct{}
	SenderChannel            chan Message
	BroadcastChannel         chan BroadcastTask
	NotifyBroadCasterChannel chan BroadcastBase
)

func InitChannels() {
	ShutdownChannel = make(chan struct{}, 1)
	SenderChannel = make(chan Message, 1000)
	NotifyBroadCasterChannel = make(chan BroadcastBase, 1)
	BroadcastChannel = make(chan BroadcastTask, 1000)
}

func CloseChannels() {
	close(ShutdownChannel)
	close(SenderChannel)
	close(NotifyBroadCasterChannel)
	close(BroadcastChannel)
}
