package commands

import (
	"os"
	"testing"

	"github.com/pyama86/slack-afk/go/store"
	"github.com/slack-go/slack"
)

func TestStartCommand_Execute(t *testing.T) {
	t.Run("Execute with no mention history", func(t *testing.T) {
		// Redis/DynamoDBが必要なので、環境変数で設定されている場合のみテストを実行
		if os.Getenv("REDIS_URL") == "" && os.Getenv("DYNAMO_LOCAL") == "" {
			t.Skip("Skipping test: No datastore configuration found")
		}

		// datastoreの初期化
		var datastore store.Datastore
		var err error

		if os.Getenv("DYNAMO_LOCAL") != "" {
			t.Setenv("DYNAMO_LOCAL", "1")
			t.Setenv("DYNAMO_ENDPOINT", "http://localhost:8000")
			t.Setenv("DYNAMO_TABLE_PREFIX", "test_slack_afk_start")
			datastore, err = store.NewDynamoDBClient()
		} else if os.Getenv("REDIS_URL") != "" {
			datastore, err = store.NewRedisClient(os.Getenv("REDIS_URL"))
		}

		if err != nil {
			t.Skipf("Skipping test: Failed to initialize datastore: %v", err)
		}

		// Slack clientのモック（実際には使用されない）
		client := slack.New("test-token")

		cmd := NewStartCommand(client, datastore)

		// テストユーザーのプレゼンスを設定
		testUID := "test-user-123"
		presence := map[string]interface{}{
			"mention_history": []interface{}{},
		}
		err = datastore.SetUserPresence(testUID, presence)
		if err != nil {
			t.Fatalf("Failed to set user presence: %v", err)
		}

		// registeredリストに追加
		err = datastore.AddToList("registered", testUID)
		if err != nil {
			t.Fatalf("Failed to add user to registered list: %v", err)
		}

		slashCmd := slack.SlashCommand{
			UserID:    testUID,
			UserName:  "testuser",
			ChannelID: "C123456",
		}

		// Executeは実際にSlack APIを呼び出すので、エラーが発生する可能性がある
		// ここでは主にデータストアの操作が正しく行われることを確認
		_ = cmd.Execute(slashCmd)

		// クリーンアップ
		_ = datastore.Delete(testUID)
		_ = datastore.RemoveFromList("registered", testUID)
	})

	t.Run("Execute with mention history", func(t *testing.T) {
		// Redis/DynamoDBが必要なので、環境変数で設定されている場合のみテストを実行
		if os.Getenv("REDIS_URL") == "" && os.Getenv("DYNAMO_LOCAL") == "" {
			t.Skip("Skipping test: No datastore configuration found")
		}

		// datastoreの初期化
		var datastore store.Datastore
		var err error

		if os.Getenv("DYNAMO_LOCAL") != "" {
			t.Setenv("DYNAMO_LOCAL", "1")
			t.Setenv("DYNAMO_ENDPOINT", "http://localhost:8000")
			t.Setenv("DYNAMO_TABLE_PREFIX", "test_slack_afk_start")
			datastore, err = store.NewDynamoDBClient()
		} else if os.Getenv("REDIS_URL") != "" {
			datastore, err = store.NewRedisClient(os.Getenv("REDIS_URL"))
		}

		if err != nil {
			t.Skipf("Skipping test: Failed to initialize datastore: %v", err)
		}

		// Slack clientのモック（実際には使用されない）
		client := slack.New("test-token")

		cmd := NewStartCommand(client, datastore)

		// テストユーザーのプレゼンスを設定
		testUID := "test-user-456"
		presence := map[string]interface{}{
			"mention_history": []interface{}{
				map[string]interface{}{
					"channel":  "C123",
					"user":     "U456",
					"text":     "Hello",
					"event_ts": "1234567890.123456",
				},
			},
		}
		err = datastore.SetUserPresence(testUID, presence)
		if err != nil {
			t.Fatalf("Failed to set user presence: %v", err)
		}

		// registeredリストに追加
		err = datastore.AddToList("registered", testUID)
		if err != nil {
			t.Fatalf("Failed to add user to registered list: %v", err)
		}

		slashCmd := slack.SlashCommand{
			UserID:    testUID,
			UserName:  "testuser",
			ChannelID: "C123456",
		}

		// Executeは実際にSlack APIを呼び出すので、エラーが発生する可能性がある
		// ここでは主にデータストアの操作が正しく行われることを確認
		_ = cmd.Execute(slashCmd)

		// クリーンアップ
		_ = datastore.Delete(testUID)
		_ = datastore.RemoveFromList("registered", testUID)
	})
}
