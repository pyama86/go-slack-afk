package commands

import (
	"fmt"
	"os"
	"strings"
)

// FormatMentionHistory formats mention history from user presence data
// Returns the mention history text and whether any mentions exist
func FormatMentionHistory(userPresence map[string]interface{}) (string, bool) {
	var mentionHistory []interface{}
	if history, ok := userPresence["mention_history"].([]interface{}); ok {
		mentionHistory = history
	}

	if len(mentionHistory) == 0 {
		return "", false
	}

	slackDomain := os.Getenv("SLACK_DOMAIN")
	if slackDomain == "" {
		slackDomain = "slack.com"
	}

	var message strings.Builder
	for _, mention := range mentionHistory {
		if m, ok := mention.(map[string]interface{}); ok {
			user, _ := m["user"].(string)
			channel, _ := m["channel"].(string)
			text, _ := m["text"].(string)
			eventTS, _ := m["event_ts"].(string)

			linkTS := eventTS
			if linkTS != "" {
				linkTS = strings.ReplaceAll(linkTS, ".", "")
			}

			mentionText := fmt.Sprintf("<@%s>: <https://%s/archives/%s/p%s|Link>\n内容: %s\n",
				user, slackDomain, channel, linkTS, text)
			message.WriteString(mentionText)
		}
	}

	return message.String(), true
}
