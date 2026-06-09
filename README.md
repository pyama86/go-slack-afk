# Slack-AFK (Go 版)

[![CI](https://github.com/pyama86/go-slack-afk/actions/workflows/ci.yml/badge.svg)](https://github.com/pyama86/go-slack-afk/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/pyama86/go-slack-afk/branch/main/graph/badge.svg)](https://codecov.io/gh/pyama86/go-slack-afk)

Slack-AFK は Slack で離席状態を管理するためのボットです。このリポジトリは Ruby 版の Slack-AFK を Golang に移植したものです。

## 機能

- `/afk [メッセージ]` - 離席状態にする
- `/lunch [メッセージ]` - ランチ中の状態にする（1 時間後に自動解除）
- `/start` - 始業状態にする
- `/finish [メッセージ]` - 退勤状態にする（翌日朝まで自動応答）
- `/comeback` - 離席状態を解除する
- `@bot-name ping` - ping に対して「pong」と応答
- `@bot-name help` - ヘルプを表示

## 特徴

- Socket Mode で動作
- リッチな応答（絵文字やブロックを使用）
- メンションの代理応答機能
- メンション履歴の記録と表示

## 必要条件

- Go 1.25.2 以上
- Redis または DynamoDB（ローカル開発には DynamoDB Local を使用可能）

## データストア

このボットは以下の2つのデータストアをサポートしています：

### Redis（デフォルト）
```bash
# .envファイルまたは環境変数
STORE_TYPE=redis
REDIS_URL=redis://localhost:6379
```

### DynamoDB
```bash
# .envファイルまたは環境変数
STORE_TYPE=dynamodb

# ローカル開発の場合
DYNAMO_LOCAL=1
DYNAMO_ENDPOINT=http://localhost:8000

# テーブル名のカスタマイズ（オプション）
DYNAMO_TABLE_PREFIX=slack_afk
```

## 環境変数

必須の環境変数：

- `SLACK_BOT_TOKEN` - Slack ボットの OAuth トークン（`xoxb-`で始まる）
- `SLACK_APP_TOKEN` - Slack アプリのトークン（`xapp-`で始まる）

ストア設定：

- `STORE_TYPE` - データストアのタイプ（`redis` または `dynamodb`、デフォルト: `redis`）
- `REDIS_URL` - Redis の URL（STORE_TYPE=redis の場合）
- `DYNAMO_LOCAL` - DynamoDB Local を使用する場合は `1` に設定
- `DYNAMO_ENDPOINT` - DynamoDB のエンドポイント（ローカル開発時）
- `DYNAMO_TABLE_PREFIX` - DynamoDB テーブル名のプレフィックス（デフォルト: `slack_afk`）

その他のオプション：

- `SLACK_DOMAIN` - Slack のドメイン（オプション、デフォルトは `slack.com`）
- `AFK_START_MESSAGE` - 始業時のカスタムメッセージ
- `AFK_FINISH_MESSAGE` - 退勤時のカスタムメッセージ
- `AFK_MENTION_THROTTLE_INTERVAL` - 同じスレッドへの AFK 自動応答を抑制する間隔（Go の `time.ParseDuration` 形式。例: `30m`, `1h`、デフォルト: `30m`）

環境変数は直接設定するか、`.env`ファイルを使用して設定できます：

```bash
# .env ファイルの例
SLACK_BOT_TOKEN=xoxb-your-token
SLACK_APP_TOKEN=xapp-your-token
REDIS_URL=redis://localhost:6379
```

`.env.sample`ファイルをコピーして`.env`ファイルを作成することもできます：

```bash
cp .env.sample .env
# 編集して適切な値を設定
```

## ローカル開発

### Docker Compose を使用

```bash
# DynamoDB Local と Redis を起動
docker-compose up -d

# DynamoDB Local のみを起動
docker-compose up -d dynamodb-local

# Redis のみを起動
docker-compose up -d redis

# DynamoDB Admin（GUIツール）を起動
docker-compose up -d dynamodb-admin
# http://localhost:8001 でアクセス可能
```

### テストを実行

```bash
# すべてのテスト（DynamoDB Local と Redis を自動起動してテスト）
make test

# Redis のみのテスト
make test-redis

# DynamoDB のみのテスト
make test-dynamodb

# 手動でテストを実行する場合
# DynamoDB のテスト
DYNAMO_LOCAL=1 go test -v ./store/... -run TestDynamoDBClient

# Redis のテスト
REDIS_URL=redis://localhost:6379 go test -v ./store/... -run TestRedisClient
```

**注意**:
- `make test` は自動的に DynamoDB Local と Redis を起動します
- Docker が起動していない場合は、外部サービスを必要とするテストはスキップされます
- 両方のデータストア実装（Redis と DynamoDB）のテストが実行されます

## ビルド方法

```bash
cd go
go build -o slack-afk
```

## 実行方法

```bash
./slack-afk
```

または、環境変数を指定して実行：

```bash
SLACK_BOT_TOKEN=xoxb-xxx SLACK_APP_TOKEN=xapp-xxx REDIS_URL=redis://localhost:6379 ./slack-afk
```

## 実装の概要

- **メインパッケージ**: アプリケーションのエントリーポイント
- **Slack パッケージ**: ソケットモードの処理
- **ハンドラーパッケージ**: コマンドとイベントの処理
- **コマンドパッケージ**: 各コマンドの実装
- **ストアパッケージ**: データストアとの連携（Redis / DynamoDB）
- **プレゼンテーションパッケージ**: リッチな応答の構築

### データストアの抽象化

ストアパッケージは `Datastore` インターフェースを定義しており、Redis と DynamoDB の両方の実装を提供しています。これにより、環境変数の設定だけで簡単にデータストアを切り替えることができます。

```go
type Datastore interface {
    Set(key string, value string) error
    SetEX(key string, value string, duration time.Duration) error
    Get(key string) (string, error)
    Delete(key string) error
    Expire(key string, duration time.Duration) error
    AddToList(key string, value string) error
    GetListRange(key string, start, stop int64) ([]string, error)
    RemoveFromList(key string, value string) error
    GetUserPresence(uid string) (map[string]interface{}, error)
    SetUserPresence(uid string, data map[string]interface{}) error
}
