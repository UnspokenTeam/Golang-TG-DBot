package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	ch "github.com/unspokenteam/golang-tg-dbot/internal/bot/app_channels"
)

var (
	token        = ""
	secretToken  = ""
	jsonifyStack = false
)

type LogEntry struct {
	Timestamp   string          `json:"ts"`
	EventType   string          `json:"event_type"`
	Level       string          `json:"level"`
	Caller      string          `json:"caller"`
	Message     string          `json:"msg"`
	EventFields json.RawMessage `json:"event_fields,omitempty"`
}

func getStructJson(structure interface{}) json.RawMessage {
	jsonData, err := json.MarshalIndent(structure, "", "\t")
	if err != nil {
		return json.RawMessage("{}")
	}
	return jsonData
}

func newLogEntry(level, message, eventType string, eventFields interface{}, isError bool) LogEntry {
	var debugStack = ""
	if !jsonifyStack && isError {
		fmt.Println(string(debug.Stack()))
	} else {
		debugStack = string(debug.Stack())
	}
	return LogEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		EventType:   eventType,
		Level:       level,
		Caller:      debugStack,
		Message:     message,
		EventFields: getStructJson(eventFields),
	}
}

func sanitizeAndFilterMessage(msg string) (string, bool) {
	//text := strings.ReplaceAll(msg, token, "token")
	//if secretToken != "" {
	//	text = strings.ReplaceAll(text, secretToken, "token")
	//}
	shouldLog := !strings.Contains(msg, "getUpdates")
	return msg, shouldLog
}

func InitLogger(botToken string, secretBotToken string, jsonifyStackSetting bool) {
	token = botToken
	secretToken = secretBotToken
	jsonifyStack = jsonifyStackSetting
}

func LogInfo(message string, eventType string, eventFields interface{}) {
	entry := newLogEntry("INFO", message, eventType, eventFields, false)
	data, _ := json.MarshalIndent(entry, "", "\t")
	fmt.Println(string(data))
}

func LogCustomLevel(level string, message string, eventType string, eventFields interface{}) {
	entry := newLogEntry(level, message, eventType, eventFields, true)
	data, _ := json.MarshalIndent(entry, "", "\t")
	fmt.Println(string(data))
}

func LogError(message string, eventType string, eventFields interface{}) {
	entry := newLogEntry("ERROR", message, eventType, eventFields, true)
	data, _ := json.MarshalIndent(entry, "", "\t")
	_, _ = fmt.Fprintln(os.Stderr, string(data))
}

func LogFatal(message string, eventType string, eventFields interface{}) {
	entry := newLogEntry("FATAL", message, eventType, eventFields, true)
	data, _ := json.MarshalIndent(entry, "", "\t")
	_, _ = fmt.Fprintln(os.Stderr, string(data))
	ch.StopChannel <- struct{}{}
}

type TelegoLogger struct{}

func (l TelegoLogger) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	LogInfo(msg, "SysLog", nil)
	return len(p), nil
}

func (l TelegoLogger) Debugf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	text, shouldLog := sanitizeAndFilterMessage(msg)
	if shouldLog {
		LogCustomLevel("DEBUG", text, "TeleGoLogger", nil)
	}
}

func (l TelegoLogger) Errorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	text, shouldLog := sanitizeAndFilterMessage(msg)
	if shouldLog {
		LogError(text, "TeleGoLogger", nil)
	}
}

func (l TelegoLogger) Debug(v ...any) {
	msg := fmt.Sprint(v...)
	text, shouldLog := sanitizeAndFilterMessage(msg)
	if shouldLog {
		LogCustomLevel("DEBUG", text, "TeleGoLogger", nil)
	}
}

func (l TelegoLogger) Info(v ...any) {
	msg := fmt.Sprint(v...)
	text, shouldLog := sanitizeAndFilterMessage(msg)
	if shouldLog {
		LogInfo(text, "TeleGoLogger", nil)
	}
}

func (l TelegoLogger) Warn(v ...any) {
	msg := fmt.Sprint(v...)
	text, shouldLog := sanitizeAndFilterMessage(msg)
	if shouldLog {
		LogCustomLevel("WARN", text, "TeleGoLogger", nil)
	}
}

func (l TelegoLogger) Error(v ...any) {
	msg := fmt.Sprint(v...)
	text, shouldLog := sanitizeAndFilterMessage(msg)
	if shouldLog {
		LogError(text, "TeleGoLogger", nil)
	}
}
