package handler_utils

import (
	"fmt"
	"github.com/mymmrac/telego"
	"net/url"
	"regexp"
	"strings"
)

var utilsBotInstance *telego.Bot

type Placeholder struct {
	ID    string
	Value string
}

func escapeMarkdownV2Base(text string) string {
	special := "_*[]()~`>#+-=|{}.!"
	var builder strings.Builder

	for _, r := range text {
		if r >= 1 && r <= 126 {
			if strings.ContainsRune(`\`+special, r) {
				builder.WriteRune('\\')
			}
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func escapeInlineCode(code string) string {
	code = strings.ReplaceAll(code, `\`, `\\`)
	code = strings.ReplaceAll(code, "`", "\\`")
	return "`" + code + "`"
}

func escapePreBlock(code string) string {
	code = strings.ReplaceAll(code, `\`, `\\`)
	code = strings.ReplaceAll(code, "`", "\\`")
	return "```\n" + code + "\n```"
}

func escapeURL(url string) string {
	url = strings.ReplaceAll(url, `\`, `\\`)
	url = strings.ReplaceAll(url, ")", "\\)")
	return url
}

func EscapeMarkdownV2Smart(input string) string {
	var placeholders []Placeholder
	placeholderCount := 0

	makePlaceholder := func(replacement string) string {
		id := fmt.Sprintf("\u0001%d\u0001", placeholderCount)
		placeholders = append(placeholders, Placeholder{id, replacement})
		placeholderCount++
		return id
	}

	rePre := regexp.MustCompile("(?s)```(.*?)```")
	input = rePre.ReplaceAllStringFunc(input, func(match string) string {
		content := rePre.FindStringSubmatch(match)[1]
		return makePlaceholder(escapePreBlock(content))
	})

	reInlineCode := regexp.MustCompile("(?s)`([^`\n]+?)`")
	input = reInlineCode.ReplaceAllStringFunc(input, func(match string) string {
		content := reInlineCode.FindStringSubmatch(match)[1]
		return makePlaceholder(escapeInlineCode(content))
	})

	reLinks := regexp.MustCompile(`\[((?:\\.|[^\[\]\\])+)\]\(((?:\\.|[^()\s])+)\)`)
	input = reLinks.ReplaceAllStringFunc(input, func(match string) string {
		matches := reLinks.FindStringSubmatch(match)
		escapedText := escapeMarkdownV2Base(matches[1])
		escapedURL := escapeURL(matches[2])
		return makePlaceholder("[" + escapedText + "](" + escapedURL + ")")
	})

	input = escapeMarkdownV2Base(input)
	for _, ph := range placeholders {
		input = strings.ReplaceAll(input, ph.ID, ph.Value)
	}

	return input
}

func MentionUser(name string, userID int64) string {
	return fmt.Sprintf("[%s](tg://user?id=%d)", name, userID)
}

func GetAddToGroupLink(text string) string {
	return fmt.Sprintf("[%s](tg://resolve?domain=%s&startgroup=true)", text, utilsBotInstance.Username())
}

func GetAddToGroupLinkWithoutHeader() string {
	return fmt.Sprintf("tg://resolve?domain=%s&startgroup=true", utilsBotInstance.Username())
}

func GetSendInviteLink(text string, inviteText string) string {
	return fmt.Sprintf(
		"[%s](tg://msg_url?url=%s&text=%s)",
		text,
		url.QueryEscape(GetAddToGroupLinkWithoutHeader()),
		strings.ReplaceAll(url.QueryEscape(inviteText), "+", "%20"),
	)
}

func InitUtils(bot *telego.Bot) {
	utilsBotInstance = bot
}

func IsGroup(upd telego.Update) bool {
	return upd.Message.From.ID != upd.Message.Chat.ID
}
