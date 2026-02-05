package slack

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pyama86/slack-afk/go/handlers"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func StartSocketModeServer(datastore store.Datastore) error {
	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	if botToken == "" {
		slog.Error("SLACK_BOT_TOKEN environment variable is not set")
		return fmt.Errorf("SLACK_BOT_TOKEN is required")
	}
	if appToken == "" {
		slog.Error("SLACK_APP_TOKEN environment variable is not set")
		return fmt.Errorf("SLACK_APP_TOKEN is required")
	}

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(api)

	commandHandler := handlers.NewCommandHandler(api, datastore)
	eventHandler := handlers.NewEventHandler(api, datastore)

	slog.Info("Starting Slack Socket Mode server")

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				client.Ack(*evt.Request)
				payload, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					slog.Error("Failed to cast event data to EventsAPIEvent")
					continue
				}

				switch payload.Type {
				case slackevents.CallbackEvent:
					innerEvent := payload.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						if err := eventHandler.HandleMention(ev); err != nil {
							slog.Error("Failed to handle mention", slog.String("user", ev.User), slog.String("channel", ev.Channel), slog.Any("error", err))
						}
					case *slackevents.MessageEvent:
						if err := eventHandler.HandleMessage(ev); err != nil {
							slog.Error("Failed to handle message", slog.String("user", ev.User), slog.String("channel", ev.Channel), slog.Any("error", err))
						}
					}
				}
			case socketmode.EventTypeSlashCommand:
				client.Ack(*evt.Request)
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					slog.Error("Failed to cast event data to SlashCommand")
					continue
				}
				commandHandler.Handle(cmd)
			case socketmode.EventTypeConnectionError:
				slog.Error("Slack Socket Mode connection error", slog.Any("data", evt.Data))
			case socketmode.EventTypeIncomingError:
				slog.Error("Slack Socket Mode incoming error", slog.Any("data", evt.Data))
			default:
				slog.Debug("Received unhandled Socket Mode event", slog.String("type", string(evt.Type)))
			}
		}
	}()

	err := client.Run()
	if err != nil {
		slog.Error("Slack Socket Mode server failed", slog.Any("error", err))
	}
	return err
}
