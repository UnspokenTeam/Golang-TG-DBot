package app_channels

var (
	StartChannel   chan struct{}
	RestartChannel chan struct{}
	StopChannel    chan struct{}
)

func InitChannels() {
	StartChannel = make(chan struct{}, 1)
	RestartChannel = make(chan struct{}, 1)
	StopChannel = make(chan struct{}, 1)
}
