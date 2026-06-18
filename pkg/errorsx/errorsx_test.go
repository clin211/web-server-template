package errorsx

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestBizErrorHTTPStatus(t *testing.T) {
	bizErr := NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	assert.Equal(t, http.StatusOK, bizErr.HTTPStatus())

	authErr := NewBizError(CodeAuthTokenInvalid, "Auth.TokenInvalid", "令牌无效")
	assert.Equal(t, http.StatusUnauthorized, authErr.HTTPStatus())

	sysErr := NewBizError(CodeDatabaseReadFailed, "Infra.DatabaseReadFailed", "数据库读取失败")
	assert.Equal(t, http.StatusInternalServerError, sysErr.HTTPStatus())
}

func TestFromErrorPreservesGRPCAuthStatus(t *testing.T) {
	authStatus := NewBizError(CodeAuthTokenInvalid, "Auth.TokenInvalid", "令牌无效").GRPCStatus().Err()
	converted := FromError(authStatus)
	assert.Equal(t, CodeAuthUnauthenticated, converted.Code)
	assert.Equal(t, "Auth.TokenInvalid", converted.Reason)
	assert.Equal(t, http.StatusUnauthorized, converted.HTTPStatus())
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
	}

	for _, tt := range tests {
		assert.Equal(t, tt.level, GetErrorLevel(tt.code))
	}
}

func TestGetHTTPCode(t *testing.T) {
	tests := []struct {
		code     BizCode
		httpCode int
	}{
		{CodeOK, http.StatusOK},
		{CodeUserNotFound, http.StatusOK},
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
	normalErr := errors.New("普通错误")
	converted = FromError(normalErr)
	assert.Equal(t, CodeInternalServer, converted.Code)
	assert.Equal(t, "Unknown", converted.Reason)
	assert.Equal(t, "普通错误", converted.Message)
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

// 基准测试
func BenchmarkNewBizError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在")
	}
}

func TestBizErrorHTTPStatusFromCode(t *testing.T) {
	tests := []struct {
		name     string
		err      *BizError
		wantHTTP int
	}{
		{
			name:     "user not found returns 200",
			err:      NewBizError(CodeUserNotFound, "User.NotFound", "用户不存在"),
			wantHTTP: http.StatusOK,
		},
		{
			name:     "auth token invalid returns 401",
			err:      NewBizError(CodeAuthTokenInvalid, "Auth.TokenInvalid", "令牌无效"),
			wantHTTP: http.StatusUnauthorized,
		},
		{
			name:     "permission denied returns 403",
			err:      NewBizError(CodeUserPermissionDenied, "Auth.PermissionDenied", "权限不足"),
			wantHTTP: http.StatusForbidden,
		},
		{
			name:     "database read failed returns 500",
			err:      NewBizError(CodeDatabaseReadFailed, "Infra.DatabaseReadFailed", "数据库读取失败"),
			wantHTTP: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantHTTP, tt.err.HTTPStatus())
		})
	}
}

func TestFromErrorConvertsPlainErrorWithoutCompat(t *testing.T) {
	converted := FromError(errors.New("普通错误"))
	assert.Equal(t, CodeInternalServer, converted.Code)
	assert.Equal(t, "Unknown", converted.Reason)
	assert.Equal(t, "普通错误", converted.Message)
}

func TestGetHTTPCodeKeepsProtocolMappings(t *testing.T) {
	tests := []struct {
		name     string
		code     BizCode
		wantHTTP int
	}{
		{name: "ok", code: CodeOK, wantHTTP: http.StatusOK},
		{name: "user not found", code: CodeUserNotFound, wantHTTP: http.StatusOK},
		{name: "unauthenticated", code: CodeAuthUnauthenticated, wantHTTP: http.StatusUnauthorized},
		{name: "token invalid", code: CodeAuthTokenInvalid, wantHTTP: http.StatusUnauthorized},
		{name: "token expired", code: CodeAuthTokenExpired, wantHTTP: http.StatusUnauthorized},
		{name: "permission denied", code: CodeUserPermissionDenied, wantHTTP: http.StatusForbidden},
		{name: "too many requests", code: CodeTooManyRequests, wantHTTP: http.StatusTooManyRequests},
		{name: "service unavailable", code: CodeServiceUnavailable, wantHTTP: http.StatusServiceUnavailable},
		{name: "request timeout", code: CodeRequestTimeout, wantHTTP: http.StatusGatewayTimeout},
		{name: "database connect failed", code: CodeDatabaseConnectFailed, wantHTTP: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantHTTP, GetHTTPCode(tt.code))
		})
	}
}

func TestFromErrorUsesUnifiedGRPCReason(t *testing.T) {
	converted := FromError(status.Error(codes.Internal, "boom"))
	assert.Equal(t, CodeInternalServer, converted.Code)
	assert.Equal(t, "GRPCError", converted.Reason)
	assert.Equal(t, "boom", converted.Message)
}

func TestPredefinedMessagesUseChinese(t *testing.T) {
	assert.Equal(t, "成功", OK.Message)
	assert.Equal(t, "内部服务器错误", ErrInternal.Message)
	assert.Equal(t, "资源未找到", ErrNotFound.Message)
	assert.Equal(t, "请求体绑定失败", ErrBind.Message)
	assert.Equal(t, "参数校验失败", ErrInvalidArgument.Message)
	assert.Equal(t, "未认证", ErrUnauthenticated.Message)
	assert.Equal(t, "权限不足", ErrPermissionDenied.Message)
	assert.Equal(t, "操作失败，请稍后重试", ErrOperationFailed.Message)
}

func TestSuccessUsesChineseDefaultMessage(t *testing.T) {
	resp := Success(nil)
	assert.Equal(t, "成功", resp.Message)
}
