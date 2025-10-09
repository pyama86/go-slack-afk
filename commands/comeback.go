package commands

import (
	"log/slog"

	"github.com/pyama86/slack-afk/go/presentation/blocks"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

// ComebackCommand handles the /comeback command
type ComebackCommand struct {
	client    *slack.Client
	datastore store.Datastore
}

// NewComebackCommand creates a new ComebackCommand
func NewComebackCommand(client *slack.Client, datastore store.Datastore) *ComebackCommand {
	return &ComebackCommand{
		client:    client,
		datastore: datastore,
	}
}

// Execute handles the /comeback command
func (c *ComebackCommand) Execute(cmd slack.SlashCommand) error {
	uid := cmd.UserID
	userName := cmd.UserName
	channelID := cmd.ChannelID

	// Get user presence
	userPresence, err := c.datastore.GetUserPresence(uid)
	if err != nil {
		slog.Error("Failed to get user presence", slog.Any("error", err))
		return err
	}

	// Post message to channel
	_, _, err = c.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks.ComebackBlocks(userName)...))
	if err != nil {
		slog.Error("Failed to post message", slog.Any("error", err))
		return err
	}

	// Remove user from datastore
	if err := c.datastore.Delete(uid); err != nil {
		slog.Error("Failed to delete user from datastore", slog.Any("error", err))
		return err
	}

	// Remove user from registered list
	if err := c.datastore.RemoveFromList("registered", uid); err != nil {
		slog.Error("Failed to remove user from registered list", slog.Any("error", err))
		return err
	}

	// Format mention history message
	mentionText, hasMentions := FormatMentionHistory(userPresence)
	var responseMessage string
	if !hasMentions {
		responseMessage = "おかえりなさい!!1特にいない間にメンションは飛んでこなかったみたいです。"
	} else {
		responseMessage = "おかえりなさい!!1\nいない間に飛んできたメンションです\n" + mentionText
	}

	// Clear mention history after displaying
	if hasMentions {
		userPresence["mention_history"] = []interface{}{}
		if err := c.datastore.SetUserPresence(uid, userPresence); err != nil {
			slog.Error("Failed to clear mention history", slog.Any("error", err))
		}
	}

	// Response message
	_, err = c.client.PostEphemeral(channelID, uid, slack.MsgOptionText(responseMessage, false))
	if err != nil {
		slog.Error("Failed to post ephemeral message", slog.Any("error", err))
		return err
	}

	return nil
}
