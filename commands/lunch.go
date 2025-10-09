package commands

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/pyama86/slack-afk/go/presentation/blocks"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

// LunchCommand handles the /lunch command
type LunchCommand struct {
	client    *slack.Client
	datastore store.Datastore
}

// NewLunchCommand creates a new LunchCommand
func NewLunchCommand(client *slack.Client, datastore store.Datastore) *LunchCommand {
	return &LunchCommand{
		client:    client,
		datastore: datastore,
	}
}

// Execute handles the /lunch command
func (c *LunchCommand) Execute(cmd slack.SlashCommand) error {
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
		message = fmt.Sprintf("%s はランチに行っています。「%s」", userName, text)
	} else {
		message = fmt.Sprintf("%s はランチに行っています。反応が遅れるかもしれません。", userName)
	}

	// Save to datastore with 1 hour expiration
	if err := c.datastore.Set(uid, message); err != nil {
		slog.Error("Failed to set message", slog.Any("error", err))
		return err
	}
	if err := c.datastore.Expire(uid, 1*time.Hour); err != nil {
		slog.Error("Failed to set expiration", slog.Any("error", err))
		return err
	}

	// Post message to channel
	_, _, err = c.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks.LunchBlocks(userName, text)...))
	if err != nil {
		slog.Error("Failed to post message", slog.Any("error", err))
		return err
	}

	// Get JST time
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)

	// Save last lunch date
	userPresence["last_lunch_date"] = now.Format(time.RFC3339)
	if err := c.datastore.SetUserPresence(uid, userPresence); err != nil {
		slog.Error("Failed to set user presence", slog.Any("error", err))
		return err
	}

	// Response message
	returnTime := now.Add(1 * time.Hour).Format("15:04")
	_, err = c.client.PostEphemeral(channelID, uid, slack.MsgOptionText(fmt.Sprintf("行ってらっしゃい!!1 %sに自動で解除します", returnTime), false))
	if err != nil {
		slog.Error("Failed to post ephemeral message", slog.Any("error", err))
		return err
	}

	return nil
}
