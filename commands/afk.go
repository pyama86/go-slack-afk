package commands

import (
	"fmt"
	"log/slog"

	"github.com/pyama86/slack-afk/go/presentation/blocks"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

// AfkCommand handles the /afk command
type AfkCommand struct {
	client    *slack.Client
	datastore store.Datastore
}

// NewAfkCommand creates a new AfkCommand
func NewAfkCommand(client *slack.Client, datastore store.Datastore) *AfkCommand {
	return &AfkCommand{
		client:    client,
		datastore: datastore,
	}
}

// Execute handles the /afk command
func (c *AfkCommand) Execute(cmd slack.SlashCommand) error {
	uid := cmd.UserID
	text := cmd.Text
	userName := cmd.UserName
	channelID := cmd.ChannelID

	// Add user to registered list
	if err := c.datastore.AddToList("registered", uid); err != nil {
		slog.Error("Failed to add user to registered list", slog.Any("error", err))
		return err
	}

	// Reset user's mention history
	userPresence, err := c.datastore.GetUserPresence(uid)
	if err != nil {
		slog.Error("Failed to get user presence", slog.Any("error", err))
		return err
	}
	userPresence["mention_history"] = []interface{}{}
	if err := c.datastore.SetUserPresence(uid, userPresence); err != nil {
		slog.Error("Failed to set user presence", slog.Any("error", err))
		return err
	}

	// Set message
	var message string
	if text != "" {
		message = fmt.Sprintf("%s は席を外しています。「%s」", userName, text)
		_, _, err := c.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks.AfkBlocks(userName, text)...))
		if err != nil {
			slog.Error("Failed to post message", slog.Any("error", err))
			return err
		}
	} else {
		message = fmt.Sprintf("%s は席を外しています。反応が遅れるかもしれません。", userName)
		_, _, err := c.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks.AfkBlocks(userName, "")...))
		if err != nil {
			slog.Error("Failed to post message", slog.Any("error", err))
			return err
		}
	}

	// Save to datastore
	if err := c.datastore.Set(uid, message); err != nil {
		slog.Error("Failed to set message", slog.Any("error", err))
		return err
	}

	// Response message
	_, err = c.client.PostEphemeral(channelID, uid, slack.MsgOptionText("行ってらっしゃい!!1", false))
	if err != nil {
		slog.Error("Failed to post ephemeral message", slog.Any("error", err))
		return err
	}

	return nil
}
