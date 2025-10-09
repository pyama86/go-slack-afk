package store

import (
	"os"
	"testing"
	"time"
)

func TestRedisClient(t *testing.T) {
	// Redisが利用可能かどうかをチェック
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	client, err := NewRedisClient(redisURL)
	if err != nil {
		t.Skipf("Skipping Redis tests: Redis not available (%v)", err)
	}

	t.Run("Set and Get", func(t *testing.T) {
		key := "test-redis-key"
		value := "test-redis-value"

		err := client.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		result, err := client.Get(key)
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}

		if result != value {
			t.Errorf("Expected %s, got %s", value, result)
		}

		// クリーンアップ
		_ = client.Delete(key)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "test-redis-delete-key"
		value := "test-value"

		err := client.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		err = client.Delete(key)
		if err != nil {
			t.Fatalf("Failed to delete key: %v", err)
		}

		_, err = client.Get(key)
		if err == nil {
			t.Error("Expected error when getting deleted key, got nil")
		}
	})

	t.Run("Expire", func(t *testing.T) {
		key := "test-redis-expire-key"
		value := "test-value"

		err := client.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		// 1秒で期限切れ
		err = client.Expire(key, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to set expiration: %v", err)
		}

		// すぐに取得できることを確認
		result, err := client.Get(key)
		if err != nil {
			t.Fatalf("Failed to get value before expiration: %v", err)
		}
		if result != value {
			t.Errorf("Expected %s, got %s", value, result)
		}

		// 2秒待って期限切れを確認
		time.Sleep(2 * time.Second)
		_, err = client.Get(key)
		if err == nil {
			t.Error("Expected error when getting expired key, got nil")
		}
	})

	t.Run("AddToList and GetListRange", func(t *testing.T) {
		listKey := "test-redis-list"

		// 既存のデータをクリーンアップ
		existingValues, _ := client.GetListRange(listKey, 0, -1)
		for _, v := range existingValues {
			_ = client.RemoveFromList(listKey, v)
		}

		// リストに値を追加
		err := client.AddToList(listKey, "value1")
		if err != nil {
			t.Fatalf("Failed to add to list: %v", err)
		}

		err = client.AddToList(listKey, "value2")
		if err != nil {
			t.Fatalf("Failed to add to list: %v", err)
		}

		err = client.AddToList(listKey, "value3")
		if err != nil {
			t.Fatalf("Failed to add to list: %v", err)
		}

		// リストを取得
		values, err := client.GetListRange(listKey, 0, -1)
		if err != nil {
			t.Fatalf("Failed to get list range: %v", err)
		}

		if len(values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(values))
		}

		// 最新のものが最初に来ることを確認
		if values[0] != "value3" {
			t.Errorf("Expected value3 as first element, got %s", values[0])
		}

		// クリーンアップ
		for _, v := range values {
			_ = client.RemoveFromList(listKey, v)
		}
	})

	t.Run("RemoveFromList", func(t *testing.T) {
		listKey := "test-redis-remove-list"

		// 既存のデータをクリーンアップ
		existingValues, _ := client.GetListRange(listKey, 0, -1)
		for _, v := range existingValues {
			_ = client.RemoveFromList(listKey, v)
		}

		// リストに値を追加
		_ = client.AddToList(listKey, "value1")
		_ = client.AddToList(listKey, "value2")

		// 値を削除
		err := client.RemoveFromList(listKey, "value1")
		if err != nil {
			t.Fatalf("Failed to remove from list: %v", err)
		}

		// リストを取得
		values, err := client.GetListRange(listKey, 0, -1)
		if err != nil {
			t.Fatalf("Failed to get list range: %v", err)
		}

		if len(values) != 1 {
			t.Errorf("Expected 1 value, got %d", len(values))
		}

		if values[0] != "value2" {
			t.Errorf("Expected value2, got %s", values[0])
		}

		// クリーンアップ
		_ = client.RemoveFromList(listKey, "value2")
	})

	t.Run("AddToList removes duplicates", func(t *testing.T) {
		listKey := "test-redis-duplicate-list"

		// 既存のデータをクリーンアップ
		existingValues, _ := client.GetListRange(listKey, 0, -1)
		for _, v := range existingValues {
			_ = client.RemoveFromList(listKey, v)
		}

		// 同じ値を2回追加
		_ = client.AddToList(listKey, "value1")
		_ = client.AddToList(listKey, "value1")

		// リストを取得
		values, err := client.GetListRange(listKey, 0, -1)
		if err != nil {
			t.Fatalf("Failed to get list range: %v", err)
		}

		// 重複が削除されていることを確認
		if len(values) != 1 {
			t.Errorf("Expected 1 value (no duplicates), got %d", len(values))
		}

		// クリーンアップ
		_ = client.RemoveFromList(listKey, "value1")
	})

	t.Run("GetUserPresence and SetUserPresence", func(t *testing.T) {
		uid := "test-redis-user"

		// 新しいユーザーのプレゼンスを取得（デフォルト値が返る）
		presence, err := client.GetUserPresence(uid)
		if err != nil {
			t.Fatalf("Failed to get user presence: %v", err)
		}

		if presence["last_active_start_time"] == nil {
			t.Error("Expected default last_active_start_time to be set")
		}

		// プレゼンスを設定
		presence["mention_history"] = []interface{}{}
		presence["custom_field"] = "test-value"

		err = client.SetUserPresence(uid, presence)
		if err != nil {
			t.Fatalf("Failed to set user presence: %v", err)
		}

		// プレゼンスを取得
		retrievedPresence, err := client.GetUserPresence(uid)
		if err != nil {
			t.Fatalf("Failed to get user presence: %v", err)
		}

		if retrievedPresence["custom_field"] != "test-value" {
			t.Errorf("Expected custom_field to be 'test-value', got %v", retrievedPresence["custom_field"])
		}

		// メンション履歴が空の配列として保存されていることを確認
		mentionHistory, ok := retrievedPresence["mention_history"].([]interface{})
		if !ok {
			t.Error("Expected mention_history to be an array")
		}
		if len(mentionHistory) != 0 {
			t.Errorf("Expected empty mention_history, got %d items", len(mentionHistory))
		}
	})

	t.Run("SetUserPresence with mention history", func(t *testing.T) {
		uid := "test-redis-user-with-mentions"

		// メンション履歴を含むプレゼンスを設定
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

		err := client.SetUserPresence(uid, presence)
		if err != nil {
			t.Fatalf("Failed to set user presence: %v", err)
		}

		// プレゼンスを取得
		retrievedPresence, err := client.GetUserPresence(uid)
		if err != nil {
			t.Fatalf("Failed to get user presence: %v", err)
		}

		mentionHistory, ok := retrievedPresence["mention_history"].([]interface{})
		if !ok {
			t.Error("Expected mention_history to be an array")
		}

		if len(mentionHistory) != 1 {
			t.Errorf("Expected 1 mention, got %d", len(mentionHistory))
		}

		mention, ok := mentionHistory[0].(map[string]interface{})
		if !ok {
			t.Error("Expected mention to be a map")
		}

		if mention["channel"] != "C123" {
			t.Errorf("Expected channel to be 'C123', got %v", mention["channel"])
		}
	})

	t.Run("UserPresence TTL expiration", func(t *testing.T) {
		uid := "test-user-ttl"

		// プレゼンス情報を設定
		presence := map[string]interface{}{
			"custom_field": "test-value",
		}
		err := client.SetUserPresence(uid, presence)
		if err != nil {
			t.Fatalf("Failed to set user presence: %v", err)
		}

		// TTLを1秒に設定（Redisキーに直接Expireを設定）
		key := uid + "-store"
		err = client.Expire(key, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to set expiration: %v", err)
		}

		// すぐに取得できることを確認
		retrievedPresence, err := client.GetUserPresence(uid)
		if err != nil {
			t.Fatalf("Failed to get user presence before expiration: %v", err)
		}
		if retrievedPresence["custom_field"] != "test-value" {
			t.Errorf("Expected custom_field to be 'test-value', got %v", retrievedPresence["custom_field"])
		}

		// 2秒待ってTTL期限切れを確認
		time.Sleep(2 * time.Second)
		expiredPresence, err := client.GetUserPresence(uid)
		if err != nil {
			t.Fatalf("Failed to get user presence after expiration: %v", err)
		}

		// 期限切れ後はデフォルト値が返される
		if expiredPresence["custom_field"] != nil {
			t.Errorf("Expected custom_field to be nil after TTL expiration, got %v", expiredPresence["custom_field"])
		}
		if expiredPresence["last_active_start_time"] == nil {
			t.Error("Expected default last_active_start_time to be set after TTL expiration")
		}
	})
}
