package logger

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func Fatal(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...), "stack", string(debug.Stack()))
	channels.ShutdownChannel <- struct{}{}
}

type TelegoLogger struct {
	logger   *slog.Logger
	replacer *strings.Replacer
}

func (t *TelegoLogger) WithReplacer(replacer *strings.Replacer) {
	t.replacer = replacer
}

func (t *TelegoLogger) Debugf(format string, args ...any) {
	t.logger.Debug(t.replacer.Replace(fmt.Sprintf(format, args...)))
}

func (t *TelegoLogger) Errorf(format string, args ...any) {
	t.logger.Error(t.replacer.Replace(fmt.Sprintf(format, args...)))
}

func SetupLogger(serviceName string) *TelegoLogger {
	var l *slog.Logger

	if utils.IsEnvProduction() {
		uptrace.ConfigureOpentelemetry(
			uptrace.WithDeploymentEnvironment(utils.GetEnv()),
			uptrace.WithDSN(os.Getenv("UPTRACE_DSN")),
			uptrace.WithServiceName(serviceName),
			uptrace.WithServiceVersion("1.0.0"),
		)

		l = otelslog.NewLogger(serviceName)
	} else {
		l = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	slog.SetDefault(l)

	return &TelegoLogger{logger: l, replacer: strings.NewReplacer()}
}
