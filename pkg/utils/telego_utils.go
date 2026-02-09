package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"golang.org/x/time/rate"
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

func parseAndReplaceLinks(input string, makePlaceholder func(string) string) string {
	var result strings.Builder
	i := 0

	for i < len(input) {
		if input[i] == '[' {
			textStart := i + 1
			textEnd := findClosingBracket(input, i)

			if textEnd != -1 && textEnd+1 < len(input) && input[textEnd+1] == '(' {
				urlStart := textEnd + 2
				urlEnd := findBalancedClosingParen(input, textEnd+1)

				if urlEnd != -1 {
					text := input[textStart:textEnd]
					url := input[urlStart:urlEnd]

					escapedText := escapeMarkdownV2Base(text)
					escapedURL := escapeURL(url)
					replacement := "[" + escapedText + "](" + escapedURL + ")"

					result.WriteString(makePlaceholder(replacement))
					i = urlEnd + 1
					continue
				}
			}
		}

		result.WriteByte(input[i])
		i++
	}

	return result.String()
}

func findClosingBracket(s string, start int) int {
	for i := start + 1; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == ']' {
			return i
		}
	}
	return -1
}

func findBalancedClosingParen(s string, start int) int {
	if s[start] != '(' {
		return -1
	}

	depth := 0
	for i := start; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}

		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
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

	input = parseAndReplaceLinks(input, makePlaceholder)

	reItalic := regexp.MustCompile(`__(.+?)__`)
	input = reItalic.ReplaceAllStringFunc(input, func(match string) string {
		content := reItalic.FindStringSubmatch(match)[1]
		escapedContent := escapeMarkdownV2Base(content)
		return makePlaceholder("_" + escapedContent + "_")
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
	if upd.Message == nil || upd.Message.From == nil {
		return false
	}
	return upd.Message.From.ID != upd.Message.Chat.ID
}

func GetReplyParams(msg *telego.Message) *telego.ReplyParameters {
	if msg == nil {
		return nil
	}
	var msgId int
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.SenderChat != nil && msg.ReplyToMessage.SenderChat.Type == telego.ChatTypeChannel {
		msgId = msg.ReplyToMessage.MessageID
	}
	return &telego.ReplyParameters{
		MessageID:                msgId,
		ChatID:                   msg.Chat.ChatID(),
		AllowSendingWithoutReply: true,
	}
}

func GetMsgSendParams(text string, msg *telego.Message) *telego.SendMessageParams {
	return tu.Message(tu.ID(msg.Chat.ID), text).
		WithReplyParameters(GetReplyParams(msg)).
		WithMessageThreadID(msg.MessageThreadID)
}

func rateLimitWait(ctx context.Context, limiter *rate.Limiter) {
	if limiter != nil {
		if limiterErr := limiter.Wait(ctx); limiterErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Sender rate limiter error: %s", limiterErr))
		}
	}
}

func muteSpammer(ctx *th.Context, message *telego.Message, cooldown int64, limiter *rate.Limiter) {
	mutePtr := false
	perms := &telego.ChatPermissions{
		CanSendMessages:       &mutePtr,
		CanSendOtherMessages:  &mutePtr,
		CanAddWebPagePreviews: &mutePtr,
	}
	until := message.Date + cooldown

	rateLimitWait(ctx, limiter)
	if err := utilsBotInstance.RestrictChatMember(ctx.Context(), &telego.RestrictChatMemberParams{
		ChatID:                        message.Chat.ChatID(),
		UserID:                        message.From.ID,
		Permissions:                   *perms,
		UntilDate:                     until,
		UseIndependentChatPermissions: true,
	}); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Restrict chat member error: %s", err), "payload", message)
	}
}

func TryMuteSpammer(ctx *th.Context, message *telego.Message, cooldown int64, limiter *rate.Limiter) {
	rateLimitWait(ctx, limiter)
	me, err := utilsBotInstance.GetMe(ctx)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Cannot get bot instance: %s", err), "payload", message)
		return
	}

	rateLimitWait(ctx, limiter)
	botChatMember, getBotErr := utilsBotInstance.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: tu.ID(message.Chat.ID), UserID: me.ID,
	})
	if getBotErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Cannot get chat bot instance: %s", getBotErr), "payload", message)
		return
	}

	allowed := false
	switch m := botChatMember.(type) {
	case *telego.ChatMemberOwner:
		allowed = true
	case *telego.ChatMemberAdministrator:
		allowed = m.CanRestrictMembers
	}

	rateLimitWait(ctx, limiter)
	memberToMute, memberGetErr := utilsBotInstance.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: tu.ID(message.Chat.ID), UserID: message.From.ID,
	})
	if memberGetErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Cannot get chat member instance: %s", memberGetErr), "payload", message)
	}
	switch memberToMute.(type) {
	case *telego.ChatMemberOwner, *telego.ChatMemberAdministrator:
		return
	}
	if message.Chat.Type != telego.ChatTypeSupergroup {
		return
	}
	if allowed {
		muteSpammer(ctx, message, cooldown, limiter)
	}
}

func IsValidUser(msg *telego.Message) bool {
	return !(msg == nil || msg.Chat.Type == telego.ChatTypeChannel || msg.IsAutomaticForward ||
		(msg.From != nil && msg.From.IsBot))
}

func IsMessageChatCommand(msg *telego.Message) bool {
	return !(msg == nil || msg.Chat.Type == telego.ChatTypeChannel || msg.Text == "" ||
		msg.Text[0] != '/' || msg.IsAutomaticForward || (msg.From != nil && msg.From.IsBot))
}

func GetChatMemberCount(ctx context.Context, chatId int64, limiter *rate.Limiter) int {
	rateLimitWait(ctx, limiter)
	count, err := utilsBotInstance.GetChatMemberCount(ctx, &telego.GetChatMemberCountParams{
		ChatID: tu.ID(chatId)})
	if err != nil || count == nil {
		slog.ErrorContext(ctx, fmt.Sprintf("GetChatMemberCount error: %s", err), "chat_id", chatId)
		return 1
	}
	return *count
}

func GetFormattedLink(header, url string) string {
	return fmt.Sprintf("[%s](%s)", header, url)
}

func GetChatLink(header, chatName string) string {
	return fmt.Sprintf("[%s](tg://resolve?domain=%s)", header, chatName)
}

func GetStrangerName(msg *telego.Message) string {
	if msg == nil {
		return ""
	}
	if msg.SenderChat != nil {
		return msg.SenderChat.Title
	} else if msg.From != nil {
		return msg.From.FirstName
	}
	return ""
}

func IsUserInChat(ctx context.Context, chatId, userId int64, limiter *rate.Limiter) bool {
	rateLimitWait(ctx, limiter)
	member, err := utilsBotInstance.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: tu.ID(chatId),
		UserID: userId,
	})
	var apiErr *ta.Error
	if err != nil && errors.As(err, &apiErr) && strings.Contains(apiErr.Description, "CHAT_ADMIN_REQUIRED") {
		return true
	}
	return err == nil && member != nil && member.MemberStatus() != "left" && member.MemberStatus() != "kicked"
}
