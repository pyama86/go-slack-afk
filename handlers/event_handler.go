package handlers

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/pyama86/slack-afk/go/presentation/blocks"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type EventHandler struct {
	client    *slack.Client
	datastore store.Datastore
}

func NewEventHandler(client *slack.Client, datastore store.Datastore) *EventHandler {
	return &EventHandler{
		client:    client,
		datastore: datastore,
	}
}

func (h *EventHandler) HandleMention(ev *slackevents.AppMentionEvent) error {
	slog.Info("Received mention", slog.String("user", ev.User), slog.String("channel", ev.Channel), slog.String("text", ev.Text))

	// Check if the message contains "ping"
	if strings.Contains(strings.ToLower(ev.Text), "ping") {
		_, _, err := h.client.PostMessage(
			ev.Channel,
			slack.MsgOptionText("pong", false),
			slack.MsgOptionTS(ev.ThreadTimeStamp),
		)
		if err != nil {
			slog.Error("Failed to post pong message", slog.Any("error", err))
			return err
		}
		return nil
	}

	// Check if the message contains "help"
	if strings.Contains(strings.ToLower(ev.Text), "help") {
		return h.HandleHelp(ev)
	}

	return nil
}

func (h *EventHandler) HandleMessage(ev *slackevents.MessageEvent) error {
	// Ignore certain message types
	if ev.SubType == "channel_join" || ev.SubType == "bot_message" {
		return nil
	}

	// Ignore certain patterns
	if matched, _ := regexp.MatchString(`\+\+|is up to [0-9]+ points!`, ev.Text); matched {
		return nil
	}

	// Get registered users
	registeredUsers, err := h.datastore.GetListRange("registered", 0, -1)
	if err != nil {
		slog.Error("Failed to get registered users", slog.Any("error", err))
		return err
	}

	// Find mentioned users
	var mentionedUsers []string
	for _, uid := range registeredUsers {
		if strings.Contains(ev.Text, "<@"+uid+">") {
			mentionedUsers = append(mentionedUsers, uid)
		}
	}

	// Process each mentioned user
	for _, uid := range mentionedUsers {
		// Get user's away message
		message, err := h.datastore.Get(uid)
		if err != nil || message == "" {
			continue
		}

		// Update user's mention history
		userPresence, err := h.datastore.GetUserPresence(uid)
		if err != nil {
			slog.Error("Failed to get user presence", slog.Any("error", err))
			continue
		}

		// Initialize mention history if needed
		if userPresence["mention_history"] == nil || isMap(userPresence["mention_history"]) {
			userPresence["mention_history"] = []interface{}{}
		}

		// Add new mention to history
		mentionHistory, ok := userPresence["mention_history"].([]interface{})
		if !ok {
			mentionHistory = []interface{}{}
		}

		// Create mention record
		mentionText := ev.Text
		if mentionText != "" {
			mentionText = strings.ReplaceAll(mentionText, "<@"+uid+">", "")
		}

		mention := map[string]interface{}{
			"channel":  ev.Channel,
			"user":     ev.User,
			"text":     mentionText,
			"event_ts": ev.TimeStamp,
		}

		// Add to history
		mentionHistory = append(mentionHistory, mention)
		userPresence["mention_history"] = mentionHistory

		// Save updated user presence
		if err := h.datastore.SetUserPresence(uid, userPresence); err != nil {
			slog.Error("Failed to set user presence", slog.Any("error", err))
			continue
		}

		// Send auto-response
		_, _, err = h.client.PostMessage(
			ev.Channel,
			slack.MsgOptionText("自動応答: "+message, false),
			slack.MsgOptionTS(ev.ThreadTimeStamp),
		)
		if err != nil {
			slog.Error("Failed to post auto-response", slog.Any("error", err))
			continue
		}
	}

	return nil
}

// Helper function to check if an interface is a map
func isMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

func (h *EventHandler) HandleHelp(ev *slackevents.AppMentionEvent) error {
	slog.Info("Received help request", slog.String("user", ev.User), slog.String("channel", ev.Channel))

	_, _, err := h.client.PostMessage(
		ev.Channel,
		slack.MsgOptionBlocks(blocks.HelpBlocks()...),
		slack.MsgOptionTS(ev.ThreadTimeStamp),
	)

	if err != nil {
		slog.Error("Failed to post help message", slog.Any("error", err))
		return err
	}

	return nil
}
