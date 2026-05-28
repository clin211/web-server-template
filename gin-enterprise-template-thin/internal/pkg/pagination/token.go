package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// PageToken 是我们在 Token 字符串中隐藏的结构
// 实际生产中，你可以加入 Salt 签名或加密，防止客户端伪造
// 使用 map 结构支持多个字段，便于扩展
type PageToken struct {
	Fields map[string]interface{} `json:"f"` // 存储多个字段的键值对
}

// Cursor 表示解析后的游标对象，提供便捷的字段访问方法
type Cursor struct {
	fields map[string]interface{}
}

// NewCursor 创建一个新的游标对象，支持传入多个 key-value 对
// 例如：NewCursor("id", 123, "created_at", 1699123456)
func NewCursor(kvPairs ...interface{}) (*Cursor, error) {
	if len(kvPairs)%2 != 0 {
		return nil, errors.New("key-value pairs must be even number")
	}

	fields := make(map[string]interface{})
	for i := 0; i < len(kvPairs); i += 2 {
		key, ok := kvPairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("key at index %d must be string", i)
		}
		fields[key] = kvPairs[i+1]
	}

	return &Cursor{fields: fields}, nil
}

// Encode 将游标编码为 base64 字符串
func (c *Cursor) Encode() (string, error) {
	if c == nil || len(c.fields) == 0 {
		return "", nil
	}
	t := PageToken{Fields: c.fields}
	b, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor: %w", err)
	}
	// 使用 URL 安全的 Base64 编码
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetInt64 获取指定 key 的 int64 值
func (c *Cursor) GetInt64(key string) (int64, bool) {
	if c == nil || c.fields == nil {
		return 0, false
	}
	val, ok := c.fields[key]
	if !ok {
		return 0, false
	}
	// 支持多种数字类型转换
	switch v := val.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	default:
		return 0, false
	}
}

// GetString 获取指定 key 的 string 值
func (c *Cursor) GetString(key string) (string, bool) {
	if c == nil || c.fields == nil {
		return "", false
	}
	val, ok := c.fields[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetFloat64 获取指定 key 的 float64 值
func (c *Cursor) GetFloat64(key string) (float64, bool) {
	if c == nil || c.fields == nil {
		return 0, false
	}
	val, ok := c.fields[key]
	if !ok {
		return 0, false
	}
	// 支持多种数字类型转换
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}

// DecodeCursor 解析 page_token 字符串，返回 Cursor 对象
func DecodeCursor(tokenStr string) (*Cursor, error) {
	if tokenStr == "" {
		return nil, nil
	}
	b, err := base64.URLEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid page_token format: %w", err)
	}
	var t PageToken
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("invalid page_token payload: %w", err)
	}
	if t.Fields == nil {
		return nil, nil
	}
	return &Cursor{fields: t.Fields}, nil
}
