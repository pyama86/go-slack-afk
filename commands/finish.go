package commands

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pyama86/slack-afk/go/presentation/blocks"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

// FinishCommand handles the /finish command
type FinishCommand struct {
	client    *slack.Client
	datastore store.Datastore
}

// NewFinishCommand creates a new FinishCommand
func NewFinishCommand(client *slack.Client, datastore store.Datastore) *FinishCommand {
	return &FinishCommand{
		client:    client,
		datastore: datastore,
	}
}

// Execute handles the /finish command
func (c *FinishCommand) Execute(cmd slack.SlashCommand) error {
	uid := cmd.UserID
	text := cmd.Text
	userName := cmd.UserName
	channelID := cmd.ChannelID

	// Add user to registered list
	if err := c.datastore.AddToList("registered", uid); err != nil {
		slog.Error("Failed to add user to registered list", slog.Any("error", err))
		return err
	}

	// Set message
	var message string
	if text != "" {
		message = fmt.Sprintf("%s は退勤しました。「%s」", userName, text)
	} else {
		message = fmt.Sprintf("%s は退勤しました。反応が遅れるかもしれません。", userName)
	}

	// Save to datastore with expiration until tomorrow morning
	if err := c.datastore.Set(uid, message); err != nil {
		slog.Error("Failed to set message", slog.Any("error", err))
		return err
	}

	// Calculate expiration time (until 9:00 AM tomorrow)
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, jst)
	expireDuration := tomorrow.Sub(now)
	if err := c.datastore.Expire(uid, expireDuration); err != nil {
		slog.Error("Failed to set expiration", slog.Any("error", err))
		return err
	}

	// Get user presence
	userPresence, err := c.datastore.GetUserPresence(uid)
	if err != nil {
		slog.Error("Failed to get user presence", slog.Any("error", err))
		return err
	}

	// Set today's end time
	userPresence["today_end"] = now.Format(time.RFC3339)
	if err := c.datastore.SetUserPresence(uid, userPresence); err != nil {
		slog.Error("Failed to set user presence", slog.Any("error", err))
		return err
	}

	// Post message to channel
	_, _, err = c.client.PostMessage(channelID, slack.MsgOptionBlocks(blocks.FinishBlocks(userName, text)...))
	if err != nil {
		slog.Error("Failed to post message", slog.Any("error", err))
		return err
	}

	// Get finish message from environment variable or use default
	finishMessage := os.Getenv("AFK_FINISH_MESSAGE")
	if finishMessage == "" {
		finishMessage = "お疲れさまでした!!1"
	}

	// Add begin time if available
	beginTimeStr, ok := userPresence["today_begin"].(string)
	if ok {
		beginTime, err := time.Parse(time.RFC3339, beginTimeStr)
		if err == nil {
			finishMessage += fmt.Sprintf("\n始業時刻:%s", beginTime.Format("15:04"))
		}
	}

	// Add auto-disable time
	finishMessage += fmt.Sprintf("\n明日の%sに自動で解除します", tomorrow.Format("15:04"))

	// Response message
	_, err = c.client.PostEphemeral(channelID, uid, slack.MsgOptionText(finishMessage, false))
	if err != nil {
		slog.Error("Failed to post ephemeral message", slog.Any("error", err))
		return err
	}

	return nil
}
