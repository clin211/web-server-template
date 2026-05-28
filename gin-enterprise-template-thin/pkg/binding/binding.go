package binding

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// 内部包变量
var (
	// originalValidator 存储原始的 Gin 验证器，用于最终验证
	originalValidator binding.StructValidator

	// initOnce 确保验证器只被捕获一次
	initOnce sync.Once
)

// init 捕获原始验证器并将全局验证器设置为 nil。
// 这在包导入时发生，且只发生一次。
func init() {
	// 使用 Once 确保即使多个 goroutine 同时进入也只执行一次
	initOnce.Do(func() {
		// 保存原始验证器
		originalValidator = binding.Validator

		// 将全局验证器设置为 nil 以在绑定期间跳过验证
		binding.Validator = nil
	})
}

// Bind 处理来自多个源的数据，不在每个步骤之间进行验证，
// 然后在最后执行一次验证。
//
// 这解决了从一个源（例如 URI）绑定失败的问题，
// 因为来自另一个源（例如 JSON）的字段尚未绑定。
//
// 示例用法：
//
//	var req UserUpdateRequest
//	if err := binding.Bind(c, &req, binding.URI, binding.JSON); err != nil {
//	    c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
//	    return
//	}
func Bind(c *gin.Context, obj interface{}, bindFuncs ...func(*gin.Context, interface{}) error) error {
	// 执行所有绑定函数（验证已被禁用）
	for _, bindFunc := range bindFuncs {
		if err := bindFunc(c, obj); err != nil {
			return err // 返回解析错误但不返回验证错误
		}
	}

	// 在所有绑定完成后手动执行验证
	if originalValidator != nil {
		return originalValidator.ValidateStruct(obj)
	}

	return nil
}

// 用于 Bind 的通用绑定函数

// URI 将 URI 参数绑定到给定对象。
// 使用 Gin 的 ShouldBindUri 但不进行验证。
func URI(c *gin.Context, obj interface{}) error {
	return c.ShouldBindUri(obj)
}

// JSON 将 JSON 请求体绑定到给定对象。
// 使用 Gin 的 ShouldBindJSON 但不进行验证。
func JSON(c *gin.Context, obj interface{}) error {
	return c.ShouldBindJSON(obj)
}
