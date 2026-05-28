package errorsx

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBizError(t *testing.T) {
	// 测试创建新的业务错误
	err := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")

	assert.Equal(t, CodeUserNotFound, err.Code)
	assert.Equal(t, LevelUser, err.Level)
	assert.Equal(t, "User.NotFound", err.Reason)
	assert.Equal(t, "用户不存在", err.Message)
	assert.Equal(t, "biz error: code=20101 reason=User.NotFound message=用户不存在", err.Error())
}

func TestBizErrorWithMethods(t *testing.T) {
	err := NewBizError(CodeUserInsufficientBalance, "User.InsufficientBalance", "用户余额不足").
		WithDetails("当前余额：0.00，需要：99.99").
		WithMetadata(map[string]interface{}{
			"current_balance": 0.00,
			"required_amount": 99.99,
			"retry_after":     300,
			"help_url":        "https://example.com/help/balance",
		})

	assert.Equal(t, "当前余额：0.00，需要：99.99", err.Details)
	assert.Equal(t, 0.00, err.Metadata["current_balance"])
	assert.Equal(t, 99.99, err.Metadata["required_amount"])
	assert.Equal(t, 300, err.Metadata["retry_after"])
	assert.Equal(t, "https://example.com/help/balance", err.Metadata["help_url"])
}

func TestBizErrorWithMessage(t *testing.T) {
	// 测试 WithMessage 方法
	originalErr := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")

	// 创建新的错误实例，修改消息
	newErr := originalErr.WithMessage("用户账户已被删除")

	// 验证新错误的消息已更新
	assert.Equal(t, "用户账户已被删除", newErr.Message)
	assert.Equal(t, originalErr.Code, newErr.Code)
	assert.Equal(t, originalErr.Reason, newErr.Reason)
	assert.Equal(t, originalErr.Level, newErr.Level)

	// 验证原错误的消息未受影响（避免内存共享）
	assert.Equal(t, "用户不存在", originalErr.Message)

	// 验证两个错误是不同的实例
	assert.NotSame(t, originalErr, newErr)
}

func TestBizErrorWithMessageAndMetadata(t *testing.T) {
	// 测试 WithMessage 方法的元数据复制
	originalErr := NewBizError(CodeUserInsufficientBalance, "User.InsufficientBalance", "用户余额不足").
		WithMetadata(map[string]interface{}{
			"current_balance": 100.00,
			"required_amount": 200.00,
		})

	// 修改消息并修改元数据
	newErr := originalErr.WithMessage("余额不足，请充值").
		WithMetadata(map[string]interface{}{
			"current_balance": 50.00, // 替换元数据
		})

	// 验证新错误的元数据（WithMessage 复制了原元数据，但后续的 WithMetadata 替换了）
	assert.Equal(t, 50.00, newErr.Metadata["current_balance"])
	assert.Nil(t, newErr.Metadata["required_amount"]) // 被 WithMetadata 替换了

	// 验证原错误的元数据未受影响
	assert.Equal(t, 100.00, originalErr.Metadata["current_balance"])
	assert.Equal(t, 200.00, originalErr.Metadata["required_amount"])
}

func TestGetErrorLevel(t *testing.T) {
	tests := []struct {
		code  BizCode
		level int
	}{
		{CodeInternalServer, LevelSystem},
		{CodeUserNotFound, LevelUser},
		{CodePostNotFound, LevelBusiness},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.level, GetErrorLevel(tt.code))
	}
}

func TestGetModule(t *testing.T) {
	tests := []struct {
		code   BizCode
		module int
	}{
		{CodeUserNotFound, ModuleUser},
		{CodePostNotFound, ModulePost},
		{CodeCommentNotFound, ModuleComment},
		{CodeAuthUnauthenticated, ModuleAuth},
		{CodeDatabaseReadFailed, ModuleDatabase},
		{CodeCacheReadFailed, ModuleCache},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.module, GetModule(tt.code))
	}
}

func TestGetHTTPCode(t *testing.T) {
	tests := []struct {
		code     BizCode
		httpCode int
	}{
		{CodeOK, http.StatusOK},
		{CodeUserNotFound, http.StatusOK},
		{CodePostPermissionDenied, http.StatusOK},
		{CodeInternalServer, http.StatusInternalServerError},
		{CodeDatabaseConnectFailed, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.httpCode, GetHTTPCode(tt.code))
	}
}

func TestAPIResponse(t *testing.T) {
	// 测试成功响应
	successResp := Success(map[string]string{"name": "test"}, "操作成功")
	assert.Equal(t, 0, successResp.Code)
	assert.Equal(t, "操作成功", successResp.Message)
	assert.Equal(t, "test", successResp.Data.(map[string]string)["name"])

	// 测试失败响应
	failResp := Failure(20101, "用户不存在")
	assert.Equal(t, 20101, failResp.Code)
	assert.Equal(t, "用户不存在", failResp.Message)
	assert.Nil(t, failResp.Data)

	// 测试添加错误原因
	failResp.WithReason("User.NotFound")
	assert.Equal(t, "User.NotFound", failResp.Reason)
}

func TestFromBizError(t *testing.T) {
	bizErr := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在").
		WithDetails("用户ID: 12345 不存在").
		WithMetadata(map[string]interface{}{"user_id": 12345})

	resp := FromBizError(bizErr)

	assert.Equal(t, int(CodeUserNotFound), resp.Code)
	assert.Equal(t, "用户不存在", resp.Message)
	assert.Nil(t, resp.Data)
	assert.Equal(t, "User.NotFound", resp.Reason)
}

func TestFromError(t *testing.T) {
	// 测试转换 BizError
	bizErr := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	converted := FromError(bizErr)
	assert.Equal(t, bizErr, converted)

	// 测试转换普通错误
	normalErr := errors.New("normal error")
	converted = FromError(normalErr)
	assert.Equal(t, CodeInternalServer, converted.Code)
	assert.Equal(t, "Unknown", converted.Reason)
	assert.Equal(t, "normal error", converted.Message)
}

func TestErrorCompatibility(t *testing.T) {
	// 测试旧版本错误的兼容性
	oldErr := &ErrorXCompat{
		Code:     http.StatusNotFound,
		Reason:   "User.NotFound",
		Message:  "User not found.",
		Metadata: map[string]string{"user_id": "12345"},
	}

	converted := FromError(oldErr)
	assert.Equal(t, CodeUserNotFound, converted.Code)
	assert.Equal(t, "User.NotFound", converted.Reason)
	assert.Equal(t, "User not found.", converted.Message)
	assert.Equal(t, "12345", converted.Metadata["user_id"])
}

func TestCode(t *testing.T) {
	// 测试成功情况
	assert.Equal(t, 0, Code(nil))

	// 测试 BizError
	bizErr := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	assert.Equal(t, int(CodeUserNotFound), Code(bizErr))

	// 测试普通错误
	normalErr := errors.New("test error")
	assert.Equal(t, int(CodeInternalServer), Code(normalErr))
}

func TestReason(t *testing.T) {
	// 测试成功情况
	assert.Equal(t, "", Reason(nil))

	// 测试 BizError
	bizErr := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	assert.Equal(t, "User.NotFound", Reason(bizErr))

	// 测试普通错误
	normalErr := errors.New("test error")
	assert.Equal(t, "InternalError", Reason(normalErr))
}

func TestErrorXCompat(t *testing.T) {
	// 测试旧版本兼容错误
	err := NewCompat(400, "InvalidInput", "Invalid input: %s", "username")

	assert.Equal(t, 400, err.Code)
	assert.Equal(t, "InvalidInput", err.Reason)
	assert.Equal(t, "Invalid input: username", err.Message)

	// 测试 WithMessage
	err.WithMessage("New message")
	assert.Equal(t, "New message", err.Message)

	// 测试 WithMetadata
	err.WithMetadata(map[string]string{"field": "username"})
	assert.Equal(t, "username", err.Metadata["field"])

	// 测试 KV
	err.KV("user_id", "12345", "trace_id", "abc")
	assert.Equal(t, "12345", err.Metadata["user_id"])
	assert.Equal(t, "abc", err.Metadata["trace_id"])
}

// 基准测试
func BenchmarkNewBizError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	}
}

func BenchmarkFromError(b *testing.B) {
	err := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	for i := 0; i < b.N; i++ {
		_ = FromError(err)
	}
}
