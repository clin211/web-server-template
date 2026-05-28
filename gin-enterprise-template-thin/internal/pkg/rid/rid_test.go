package rid_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/clin211/gin-enterprise-template/internal/pkg/rid"
)

// Mock Salt 函数用于测试
func Salt() string {
	return "staticSalt"
}

func TestResourceID_String(t *testing.T) {
	// 测试将 UserID 转换为字符串
	rid := rid.UserID
	assert.Equal(t, "user", rid.String(), "UserID.String() should return 'user'")
}

func TestResourceID_New(t *testing.T) {
	// 测试生成的 ID 是否具有正确的前缀
	rid := rid.UserID
	uniqueID := rid.New(1)

	assert.True(t, len(uniqueID) > 0, "Generated ID should not be empty")
	assert.Contains(t, uniqueID, "user-", "生成的 ID 应以 'user-' 前缀开头")

	// 生成另一个唯一标识符以确保唯一性
	anotherID := rid.New(2)
	assert.NotEqual(t, uniqueID, anotherID, "Generated IDs should be unique")
}

func BenchmarkResourceID_New(b *testing.B) {
	// 性能基准测试
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rid := rid.UserID
		_ = rid.New(uint64(i))
	}
}

func FuzzResourceID_New(f *testing.F) {
	// 添加预设测试数据
	f.Add(uint64(1))      // 添加种子值 `counter` 为 1
	f.Add(uint64(123456)) // 添加更大的种子值

	f.Fuzz(func(t *testing.T, counter uint64) {
		// 测试 UserID 的 New 方法
		result := rid.UserID.New(counter)

		// 断言结果不为空
		assert.NotEmpty(t, result, "生成的唯一标识符不应为空")

		// 断言结果包含正确的资源标识符前缀
		assert.Contains(t, result, rid.UserID.String()+"-", "生成的唯一标识符应包含正确的前缀")

		// 断言前缀不与 uniqueStr 部分重叠
		splitParts := strings.SplitN(result, "-", 2)
		assert.Equal(t, rid.UserID.String(), splitParts[0], "结果的前缀部分应正确匹配 UserID")

		// 断言生成的 ID 具有固定长度（基于 NewCode 配置）
		if len(splitParts) == 2 {
			assert.Equal(t, 6, len(splitParts[1]), "唯一标识符部分的长度应为 6")
		} else {
			t.Errorf("生成的唯一标识符的格式不符合预期")
		}
	})
}
