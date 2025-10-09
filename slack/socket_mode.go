package slack

import (
	"log/slog"
	"os"

	"github.com/pyama86/slack-afk/go/handlers"
	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func StartSocketModeServer(datastore store.Datastore) error {
	api := slack.New(
		os.Getenv("SLACK_BOT_TOKEN"),
		slack.OptionAppLevelToken(os.Getenv("SLACK_APP_TOKEN")),
	)

	client := socketmode.New(api)

	commandHandler := handlers.NewCommandHandler(api, datastore)
	eventHandler := handlers.NewEventHandler(api, datastore)

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				client.Ack(*evt.Request)
				payload, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}

				switch payload.Type {
				case slackevents.CallbackEvent:
					innerEvent := payload.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						if err := eventHandler.HandleMention(ev); err != nil {
							slog.Error("Failed to handle mention", slog.Any("error", err))
						}
					case *slackevents.MessageEvent:
						if err := eventHandler.HandleMessage(ev); err != nil {
							slog.Error("Failed to handle message", slog.Any("error", err))
						}
					}
				}
			case socketmode.EventTypeSlashCommand:
				client.Ack(*evt.Request)
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					continue
				}
				commandHandler.Handle(cmd)
			}
		}
	}()

	return client.Run()
}
