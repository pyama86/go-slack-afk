package store

import (
	"time"
)

// Datastore はデータストアの共通インターフェースを定義
type Datastore interface {
	// Set は単一のキー・バリューを設定する
	Set(key string, value string) error

	// Get は指定されたキーの値を取得する
	Get(key string) (string, error)

	// Expire はキーに有効期限を設定する
	Expire(key string, duration time.Duration) error

	// Delete は指定されたキーを削除する
	Delete(key string) error

	// AddToList はリストに値を追加する（重複は削除してから追加）
	AddToList(key string, value string) error

	// GetListRange はリストの指定範囲を取得する
	GetListRange(key string, start, stop int64) ([]string, error)

	// RemoveFromList はリストから値を削除する
	RemoveFromList(key string, value string) error

	// GetUserPresence はユーザーのプレゼンス情報を取得する
	GetUserPresence(uid string) (map[string]interface{}, error)

	// SetUserPresence はユーザーのプレゼンス情報を設定する
	SetUserPresence(uid string, data map[string]interface{}) error
}
