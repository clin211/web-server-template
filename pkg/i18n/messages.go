package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed default_messages.json
var defaultMessagesJSON []byte

var defaultMessages = sync.OnceValue(func() map[string]string {
	var messages map[string]string
	if err := json.Unmarshal(defaultMessagesJSON, &messages); err != nil {
		// The catalog is a build-time asset; malformed JSON is a programming error.
		panic(fmt.Errorf("decode embedded default messages: %w", err))
	}
	return messages
})

// DefaultMessage returns the embedded default-language message or its ID if unknown.
func DefaultMessage(id string) string {
	if message, ok := defaultMessages()[id]; ok {
		return message
	}
	return id
}
