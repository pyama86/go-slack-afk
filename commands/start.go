package commands

import (
	"log/slog"
	"os"
	"time"

	"github.com/pyama86/slack-afk/go/presentation/blocks"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

// StartCommand handles the /start command
type StartCommand struct {
	client    *slack.Client
	datastore store.Datastore
}

// NewStartCommand creates a new StartCommand
func NewStartCommand(client *slack.Client, datastore store.Datastore) *StartCommand {
	return &StartCommand{
		client:    client,
		datastore: datastore,
	}
}

// Execute handles the /start command
func (c *StartCommand) Execute(cmd slack.SlashCommand) error {
	uid := cmd.UserID
	userName := cmd.UserName
	channelID := cmd.ChannelID

	// Remove user from registered list
	if err := c.datastore.RemoveFromList("registered", uid); err != nil {
		slog.Error("Failed to remove user from registered list", slog.Any("error", err))
		return err
	}

	// Get user presence
	userPresence, err := c.datastore.GetUserPresence(uid)
	if err != nil {
		slog.Error("Failed to get user presence", slog.Any("error", err))
		return err
	}

	// Set today's begin time
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	userPresence["today_begin"] = now.Format(time.RFC3339)
	if err := c.datastore.SetUserPresence(uid, userPresence); err != nil {
		slog.Error("Failed to set user presence", slog.Any("error", err))
		return err
	}

	// Post message to channel
	_, _, err = c.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks.StartBlocks(userName)...))
	if err != nil {
		slog.Error("Failed to post message", slog.Any("error", err))
		return err
	}

	// Get start message from environment variable or use default
	startMessage := os.Getenv("AFK_START_MESSAGE")
	if startMessage == "" {
		startMessage = "おはようございます、今日も自分史上最高の日にしましょう!!1"
	}

	// Response message
	_, err = c.client.PostEphemeral(channelID, uid, slack.MsgOptionText(startMessage, false))
	if err != nil {
		slog.Error("Failed to post ephemeral message", slog.Any("error", err))
		return err
	}

	return nil
}
