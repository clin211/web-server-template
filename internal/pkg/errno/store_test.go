package errno

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestFromStore 表格驱动验证数据访问错误到业务错误的翻译规则。
func TestFromStore(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "nil 原样返回", err: nil, wantErr: nil},
		{name: "记录不存在翻译为 ErrNotFound", err: gorm.ErrRecordNotFound, wantErr: ErrNotFound},
		{
			name:    "包装链中的记录不存在仍可识别",
			err:     fmt.Errorf("load user: %w", gorm.ErrRecordNotFound),
			wantErr: ErrNotFound,
		},
		{
			name:    "其他数据库错误原样透传",
			err:     assert.AnError,
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromStore(tt.err)
			// nil 与同一实例均通过 Same/ErrorIs 语义判定
			if tt.wantErr == nil {
				assert.NoError(t, got)
				return
			}
			assert.True(t, errors.Is(got, tt.wantErr), "FromStore(%v) = %v, want %v", tt.err, got, tt.wantErr)
		})
	}
}
