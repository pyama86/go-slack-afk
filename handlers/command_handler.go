package handlers

import (
	"log/slog"

	"github.com/pyama86/slack-afk/go/commands"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

type CommandHandler struct {
	client    *slack.Client
	datastore store.Datastore
	commands  map[string]commands.Command
}

func NewCommandHandler(client *slack.Client, datastore store.Datastore) *CommandHandler {
	h := &CommandHandler{
		client:    client,
		datastore: datastore,
		commands:  make(map[string]commands.Command),
	}

	h.commands["/afk"] = commands.NewAfkCommand(client, datastore)
	h.commands["/lunch"] = commands.NewLunchCommand(client, datastore)
	h.commands["/start"] = commands.NewStartCommand(client, datastore)
	h.commands["/finish"] = commands.NewFinishCommand(client, datastore)
	h.commands["/comeback"] = commands.NewComebackCommand(client, datastore)

	return h
}

func (h *CommandHandler) Handle(cmd slack.SlashCommand) {
	slog.Info("Received command", slog.String("command", cmd.Command), slog.String("user", cmd.UserName), slog.String("channel", cmd.ChannelID), slog.String("text", cmd.Text))

	if command, ok := h.commands[cmd.Command]; ok {
		if err := command.Execute(cmd); err != nil {
			slog.Error("Failed to execute command",
				slog.String("command", cmd.Command),
				slog.String("user", cmd.UserName),
				slog.String("channel", cmd.ChannelID),
				slog.String("text", cmd.Text),
				slog.Any("error", err))
			if _, err := h.client.PostEphemeral(cmd.ChannelID, cmd.UserID, slack.MsgOptionText("Failed to execute command: "+err.Error(), false)); err != nil {
				slog.Error("Failed to post ephemeral error message",
					slog.String("command", cmd.Command),
					slog.String("channel", cmd.ChannelID),
					slog.String("user", cmd.UserID),
					slog.Any("error", err))
			}
		}
	} else {
		slog.Info("Unknown command", slog.String("command", cmd.Command), slog.String("user", cmd.UserName))
		if _, err := h.client.PostEphemeral(cmd.ChannelID, cmd.UserID, slack.MsgOptionText("Unknown command: "+cmd.Command, false)); err != nil {
			slog.Error("Failed to post unknown command ephemeral message",
				slog.String("command", cmd.Command),
				slog.String("channel", cmd.ChannelID),
				slog.String("user", cmd.UserID),
				slog.Any("error", err))
		}
	}
}
