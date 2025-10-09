package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient struct {
	db                *dynamodb.Client
	kvTableName       string
	listTableName     string
	presenceTableName string
}

// DynamoDBClientがDatastoreインターフェースを実装していることを保証
var _ Datastore = (*DynamoDBClient)(nil)

const (
	defaultTablePrefix = "slack_afk"
	waitInterval       = 2 * time.Second
	maxRetries         = 30
)

// NewDynamoDBClient は新しいDynamoDBクライアントを作成する
func NewDynamoDBClient() (*DynamoDBClient, error) {
	tablePrefix := os.Getenv("DYNAMO_TABLE_PREFIX")
	if tablePrefix == "" {
		tablePrefix = defaultTablePrefix
	}

	kvTableName := tablePrefix + "_kv"
	listTableName := tablePrefix + "_list"
	presenceTableName := tablePrefix + "_presence"

	if os.Getenv("DYNAMO_KV_TABLE_NAME") != "" {
		kvTableName = os.Getenv("DYNAMO_KV_TABLE_NAME")
	}
	if os.Getenv("DYNAMO_LIST_TABLE_NAME") != "" {
		listTableName = os.Getenv("DYNAMO_LIST_TABLE_NAME")
	}
	if os.Getenv("DYNAMO_PRESENCE_TABLE_NAME") != "" {
		presenceTableName = os.Getenv("DYNAMO_PRESENCE_TABLE_NAME")
	}

	var db *dynamodb.Client
	if os.Getenv("DYNAMO_LOCAL") != "" {
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion("dummy"),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %v", err)
		}

		endpoint := os.Getenv("DYNAMO_ENDPOINT")
		if endpoint == "" {
			endpoint = "http://localhost:8000"
		}

		db = dynamodb.NewFromConfig(cfg,
			func(o *dynamodb.Options) {
				o.BaseEndpoint = aws.String(endpoint)
			},
		)
	} else {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %v", err)
		}
		db = dynamodb.NewFromConfig(cfg)
	}

	client := &DynamoDBClient{
		db:                db,
		kvTableName:       kvTableName,
		listTableName:     listTableName,
		presenceTableName: presenceTableName,
	}

	if os.Getenv("DYNAMO_LOCAL") != "" {
		if err := client.EnsureTables(); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// EnsureTables はテーブルが存在することを確認し、存在しない場合は作成する
func (d *DynamoDBClient) EnsureTables() error {
	tables := []struct {
		name       string
		createFunc func(string) error
	}{
		{d.kvTableName, d.createKVTable},
		{d.listTableName, d.createListTable},
		{d.presenceTableName, d.createPresenceTable},
	}

	for _, table := range tables {
		if err := d.ensureSingleTable(table.name, table.createFunc); err != nil {
			return fmt.Errorf("failed to ensure table %s: %v", table.name, err)
		}
	}

	return nil
}

func (d *DynamoDBClient) ensureSingleTable(tableName string, createFunc func(string) error) error {
	_, err := d.db.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err == nil {
		return nil
	}

	if err := createFunc(tableName); err != nil {
		return err
	}

	for i := 0; i < maxRetries; i++ {
		out, err := d.db.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return fmt.Errorf("failed to describe table %s: %v", tableName, err)
		}

		if out.Table.TableStatus == types.TableStatusActive {
			return nil
		}

		time.Sleep(waitInterval)
	}

	return fmt.Errorf("table %s creation timed out", tableName)
}

func (d *DynamoDBClient) createKVTable(tableName string) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("key"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("key"), KeyType: types.KeyTypeHash},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := d.db.CreateTable(context.TODO(), input)
	if err != nil {
		// テーブルが既に存在する場合は無視
		if isResourceInUseException(err) {
			return nil
		}
		return fmt.Errorf("failed to create KV table: %v", err)
	}
	return nil
}

func (d *DynamoDBClient) createListTable(tableName string) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("list_key"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("item_value"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("list_key"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("item_value"), KeyType: types.KeyTypeRange},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := d.db.CreateTable(context.TODO(), input)
	if err != nil {
		// テーブルが既に存在する場合は無視
		if isResourceInUseException(err) {
			return nil
		}
		return fmt.Errorf("failed to create list table: %v", err)
	}
	return nil
}

func (d *DynamoDBClient) createPresenceTable(tableName string) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("user_id"), KeyType: types.KeyTypeHash},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := d.db.CreateTable(context.TODO(), input)
	if err != nil {
		// テーブルが既に存在する場合は無視
		if isResourceInUseException(err) {
			return nil
		}
		return fmt.Errorf("failed to create presence table: %v", err)
	}
	return nil
}

// isResourceInUseException はエラーがResourceInUseExceptionかどうかを判定
func isResourceInUseException(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "ResourceInUseException") ||
		strings.Contains(err.Error(), "preexisting table")
}

// Set は単一のキー・バリューを設定する
func (d *DynamoDBClient) Set(key string, value string) error {
	item := map[string]types.AttributeValue{
		"key":   &types.AttributeValueMemberS{Value: key},
		"value": &types.AttributeValueMemberS{Value: value},
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.kvTableName),
		Item:      item,
	}

	_, err := d.db.PutItem(context.TODO(), input)
	return err
}

// Get は指定されたキーの値を取得する
func (d *DynamoDBClient) Get(key string) (string, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.kvTableName),
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{Value: key},
		},
	}

	result, err := d.db.GetItem(context.TODO(), input)
	if err != nil {
		return "", err
	}

	if result.Item == nil {
		return "", fmt.Errorf("key not found: %s", key)
	}

	if ttlAttr, ok := result.Item["ttl"]; ok {
		if ttlVal, ok := ttlAttr.(*types.AttributeValueMemberN); ok {
			ttl, err := strconv.ParseInt(ttlVal.Value, 10, 64)
			if err == nil && time.Now().Unix() > ttl {
				// TTLが期限切れの場合は削除して not found を返す
				_ = d.Delete(key)
				return "", fmt.Errorf("key not found: %s", key)
			}
		}
	}

	if v, ok := result.Item["value"].(*types.AttributeValueMemberS); ok {
		return v.Value, nil
	}

	return "", fmt.Errorf("invalid value type for key: %s", key)
}

// Expire はキーに有効期限を設定する
func (d *DynamoDBClient) Expire(key string, duration time.Duration) error {
	ttl := time.Now().Add(duration).Unix()

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(d.kvTableName),
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{Value: key},
		},
		UpdateExpression: aws.String("SET #ttl = :ttl"),
		ExpressionAttributeNames: map[string]string{
			"#ttl": "ttl",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ttl": &types.AttributeValueMemberN{Value: strconv.FormatInt(ttl, 10)},
		},
	}

	_, err := d.db.UpdateItem(context.TODO(), input)
	return err
}

// Delete は指定されたキーを削除する
func (d *DynamoDBClient) Delete(key string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(d.kvTableName),
		Key: map[string]types.AttributeValue{
			"key": &types.AttributeValueMemberS{Value: key},
		},
	}

	_, err := d.db.DeleteItem(context.TODO(), input)
	return err
}

// AddToList はリストに値を追加する（重複は削除してから追加）
func (d *DynamoDBClient) AddToList(key string, value string) error {
	// 既存のアイテムがあれば削除
	_ = d.RemoveFromList(key, value)

	// 新しいアイテムを追加
	timestamp := time.Now().UnixNano()
	item := map[string]types.AttributeValue{
		"list_key":   &types.AttributeValueMemberS{Value: key},
		"item_value": &types.AttributeValueMemberS{Value: value},
		"timestamp":  &types.AttributeValueMemberN{Value: strconv.FormatInt(timestamp, 10)},
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.listTableName),
		Item:      item,
	}

	_, err := d.db.PutItem(context.TODO(), input)
	return err
}

// GetListRange はリストの指定範囲を取得する
func (d *DynamoDBClient) GetListRange(key string, start, stop int64) ([]string, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(d.listTableName),
		KeyConditionExpression: aws.String("list_key = :list_key"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":list_key": &types.AttributeValueMemberS{Value: key},
		},
		ScanIndexForward: aws.Bool(false), // 降順（最新から）
	}

	result, err := d.db.Query(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	var values []string
	for _, item := range result.Items {
		if v, ok := item["item_value"].(*types.AttributeValueMemberS); ok {
			values = append(values, v.Value)
		}
	}

	// start と stop でスライス
	// Redisと互換性を持たせるため、負の値をサポート
	length := int64(len(values))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop || start >= length {
		return []string{}, nil
	}

	return values[start : stop+1], nil
}

// RemoveFromList はリストから値を削除する
func (d *DynamoDBClient) RemoveFromList(key string, value string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(d.listTableName),
		Key: map[string]types.AttributeValue{
			"list_key":   &types.AttributeValueMemberS{Value: key},
			"item_value": &types.AttributeValueMemberS{Value: value},
		},
	}

	_, err := d.db.DeleteItem(context.TODO(), input)
	return err
}

// GetUserPresence はユーザーのプレゼンス情報を取得する
func (d *DynamoDBClient) GetUserPresence(uid string) (map[string]interface{}, error) {
	key := uid + "-store"
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.presenceTableName),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: key},
		},
	}

	result, err := d.db.GetItem(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		jst, _ := time.LoadLocation("Asia/Tokyo")
		now := time.Now().In(jst)
		return map[string]interface{}{
			"last_active_start_time": now.Format(time.RFC3339),
		}, nil
	}

	if v, ok := result.Item["presence_data"].(*types.AttributeValueMemberS); ok {
		var presenceData map[string]interface{}
		if err := json.Unmarshal([]byte(v.Value), &presenceData); err != nil {
			return nil, err
		}
		return presenceData, nil
	}

	return nil, fmt.Errorf("invalid presence data for user: %s", uid)
}

// SetUserPresence はユーザーのプレゼンス情報を設定する
func (d *DynamoDBClient) SetUserPresence(uid string, data map[string]interface{}) error {
	key := uid + "-store"
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ttl := time.Now().Add(30 * 24 * time.Hour).Unix()

	item := map[string]types.AttributeValue{
		"user_id":       &types.AttributeValueMemberS{Value: key},
		"presence_data": &types.AttributeValueMemberS{Value: string(jsonData)},
		"ttl":           &types.AttributeValueMemberN{Value: strconv.FormatInt(ttl, 10)},
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(d.presenceTableName),
		Item:      item,
	}

	_, err = d.db.PutItem(context.TODO(), input)
	return err
}
