package errorsx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"

	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BizCode 表示业务错误码，格式：LLMMMNNN
// LL: 错误级别（1-5）
// MMM: 业务模块（001-999）
// NNN: 具体错误（001-999）
type BizCode int32

// 错误级别定义
const (
	LevelSystem     = 1 // 系统级错误，需要开发介入
	LevelUser       = 2 // 用户操作错误，可提示用户
	LevelBusiness   = 3 // 业务逻辑错误，正常的业务流程
	LevelUpstream   = 4 // 上游服务错误
	LevelDownstream = 5 // 下游服务错误
)

// 常用错误码定义
const (
	CodeOK                      BizCode = 0     // 成功
	CodeUserNotFound            BizCode = 20101 // 用户不存在 (Level=2, Module=01, Error=01)
	CodeUserAlreadyExists       BizCode = 20102 // 用户已存在
	CodeUserInvalidCredentials  BizCode = 20103 // 用户名或密码错误
	CodeUserInsufficientBalance BizCode = 20104 // 用户余额不足
	CodeUserInvalidUsername     BizCode = 20105 // 用户名无效
	CodeUserInvalidPassword     BizCode = 20106 // 密码无效
	CodeUserPermissionDenied    BizCode = 20107 // 用户权限不足

	CodeAuthUnauthenticated BizCode = 50101 // 未认证 (Level=5, Module=01, Error=01)
	CodeAuthTokenInvalid    BizCode = 50102 // Token无效
	CodeAuthTokenExpired    BizCode = 50103 // Token过期
	CodeAuthSignToken       BizCode = 50104 // Token签名失败

	CodeDatabaseConnectFailed BizCode = 60101 // 数据库连接失败 (Level=6, Module=01, Error=01)
	CodeDatabaseReadFailed    BizCode = 60102 // 数据库读取失败
	CodeDatabaseWriteFailed   BizCode = 60103 // 数据库写入失败

	CodeCacheConnectFailed BizCode = 70101 // 缓存连接失败 (Level=7, Module=01, Error=01)
	CodeCacheReadFailed    BizCode = 70102 // 缓存读取失败
	CodeCacheWriteFailed   BizCode = 70103 // 缓存写入失败

	CodeInternalServer     BizCode = 10101 // 内部服务器错误 (Level=1, Module=01, Error=01)
	CodeServiceUnavailable BizCode = 10102 // 服务不可用
	CodeRequestTimeout     BizCode = 10103 // 请求超时
	CodeTooManyRequests    BizCode = 10104 // 请求过于频繁
)

// BizError 业务错误结构
type BizError struct {
	// 业务错误码
	Code BizCode `json:"code"`

	// 错误级别
	Level int `json:"level"`

	// 错误原因（英文，用于日志和监控）
	Reason string `json:"reason,omitempty"`

	// 用户友好消息，HTTP 输出时按请求语言翻译。
	Message string `json:"message,omitempty"`

	// MessageID 指定翻译键；为空时使用 Reason，不参与响应序列化。
	MessageID string `json:"-"`

	messageOverride bool
	messageParams   i18n.Params

	// 详细信息
	Details string `json:"details,omitempty"`

	// 元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// APIResponse 标准API响应结构
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ResponseDetail 响应详情（用于替代 ErrorDetail）
type ResponseDetail struct {
	Details  string         `json:"details,omitempty"`
	StackID  string         `json:"stack_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Headers 常用的HTTP Headers
const (
	HeaderRequestID    = "X-Request-ID"
	HeaderTimestamp    = "X-Timestamp"
	HeaderResponseTime = "X-Response-Time"
	HeaderServerID     = "X-Server-ID"
	HeaderTraceID      = "X-Trace-ID"
)

// GetErrorLevel 从错误码中提取错误级别
func GetErrorLevel(code BizCode) int {
	if code == 0 {
		return 0
	}
	return int(code/10000) % 10
}

// GetHTTPCode 根据错误码映射对应的HTTP状态码。
// 语义遵循 REST：资源不存在 404、冲突 409、请求侧参数或状态问题 4xx；
// 未显式归类的用户/业务级错误统一 400，系统与上下游错误返回 5xx/502。
// 由此保证"非 2xx 即业务失败"，与 OpenAPI 契约中 default 响应（APIError）的语义一致。
func GetHTTPCode(code BizCode) int {
	if httpCode, ok := businessHTTPCodes[code]; ok {
		return httpCode
	}

	level := GetErrorLevel(code)
	switch level {
	case LevelSystem:
		return http.StatusInternalServerError
	case LevelUpstream, LevelDownstream:
		return http.StatusBadGateway
	case LevelUser, LevelBusiness:
		// 未显式归类的用户/业务错误属于请求上下文问题，返回 400 而非 200，
		// 避免客户端把业务失败误判为成功。
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// NewBizError 创建新的业务错误
func NewBizError(code BizCode, reason, message string) *BizError {
	return &BizError{
		Code:    code,
		Level:   GetErrorLevel(code),
		Reason:  reason,
		Message: message,
	}
}

// clone 返回错误的副本（含独立的元数据映射），是所有 WithXxx 方法共享的复制语义
// 基础。预定义错误是包级共享的哨兵值，原地修改会在并发请求间产生数据竞争并
// 互相污染（例如把上一个请求的绑定错误详情带进下一个请求的日志），因此任何
// 派生操作都必须返回新实例。
func (err *BizError) clone() *BizError {
	newErr := *err
	newErr.Metadata = maps.Clone(err.Metadata)
	newErr.messageParams = maps.Clone(err.messageParams)
	return &newErr
}

// WithDetails 设置错误详情（返回新实例，不修改接收者）
func (err *BizError) WithDetails(details string) *BizError {
	newErr := err.clone()
	newErr.Details = details
	return newErr
}

// WithMetadata 设置元数据（返回新实例，不修改接收者）
func (err *BizError) WithMetadata(metadata map[string]any) *BizError {
	newErr := err.clone()
	newErr.Metadata = metadata
	return newErr
}

// WithMessage 设置错误消息（返回新实例，不修改接收者）
func (err *BizError) WithMessage(message string) *BizError {
	newErr := err.clone()
	newErr.Message = message
	newErr.MessageID = ""
	newErr.messageParams = nil
	newErr.messageOverride = true
	return newErr
}

// WithMessageID 设置翻译键并清空旧模板参数（返回新实例），保留业务码、Reason 及回退文案。
func (err *BizError) WithMessageID(id string) *BizError {
	newErr := err.clone()
	newErr.MessageID = id
	newErr.messageParams = nil
	newErr.messageOverride = false
	return newErr
}

// WithMessageParams 替换模板参数并复制参数表（返回新实例），nil 清空参数。
// 在 WithMessageID 之后调用；参数只用于翻译，不会作为元数据暴露给客户端。
// 参数值仅支持 i18n.Params 约定的内置标量，调用返回后修改原表不会影响此错误。
func (err *BizError) WithMessageParams(params i18n.Params) *BizError {
	newErr := err.clone()
	newErr.messageParams = maps.Clone(params)
	return newErr
}

// Localize 返回按请求语言翻译的副本，避免修改多个请求共享的预定义错误。
// WithMessage 显式设置的文本优先于翻译，直到再次调用 WithMessageID。
func (err *BizError) Localize(ctx context.Context) *BizError {
	newErr := err.clone()
	if newErr.messageOverride {
		return newErr
	}
	id := newErr.MessageID
	if id == "" {
		id = newErr.Reason
	}
	translator := i18n.FromContext(ctx)
	message, renderErr := localizeMessage(ctx, translator, id, newErr.messageParams)
	if renderErr == nil {
		newErr.Message = message
		return newErr
	}
	// 普通错误文案（包括上游错误）不是受信任模板，不能传给 DefaultMessage.Other。
	if id != newErr.Reason {
		if fallback, fallbackErr := localizeMessage(ctx, translator, newErr.Reason, nil); fallbackErr == nil {
			newErr.Message = fallback
		}
	}
	if newErr.Message == "" {
		newErr.Message = id
	}
	return newErr
}

func localizeMessage(ctx context.Context, translator *i18n.I18n, id string, params i18n.Params) (string, error) {
	message := &goi18n.Message{ID: id}
	result, err := translator.Localize(message, params)
	if err == nil {
		return result, nil
	}
	var missing *goi18n.MessageNotFoundErr
	if !errors.As(err, &missing) {
		// 模板执行错误可能包含输入值，诊断只记录消息键、语言和请求上下文。
		slog.WarnContext(ctx, "localize business error failed", "message_id", id, "language", translator.Language().String())
	}
	if translator.Language() != translator.DefaultLanguage() {
		return translator.Select(translator.DefaultLanguage()).Localize(message, params)
	}
	return "", err
}

// Error 实现 error 接口
func (err *BizError) Error() string {
	return fmt.Sprintf("biz error: code=%d reason=%s message=%s", err.Code, err.Reason, err.Message)
}

// Is 判断错误是否匹配
func (err *BizError) Is(target error) bool {
	if bizErr, ok := target.(*BizError); ok {
		return bizErr.Code == err.Code
	}
	return false
}

// HTTPStatus 返回该错误的 HTTP 状态码。
func (err *BizError) HTTPStatus() int {
	return GetHTTPCode(err.Code)
}

// ToGRPCCode converts HTTP status code to gRPC status code
func ToGRPCCode(code int) codes.Code {
	if grpcCode, ok := httpGRPCCodes[code]; ok {
		return grpcCode
	}
	return codes.Unknown
}

// FromGRPCCode converts gRPC status code to HTTP status code
func FromGRPCCode(code codes.Code) int {
	if httpCode, ok := grpcHTTPCodes[code]; ok {
		return httpCode
	}
	return http.StatusInternalServerError
}

// Success 创建成功响应。
// 默认 message 与 core.WriteResponse 的成功输出保持同一口径（"success"）。
func Success(data any, message ...string) *APIResponse {
	msg := "success"
	if len(message) > 0 {
		msg = message[0]
	}
	return &APIResponse{
		Code:    0,
		Message: msg,
		Data:    data,
	}
}

// Failure 创建失败响应
func Failure(code int, message string) *APIResponse {
	return &APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// WithReason 添加错误原因
func (resp *APIResponse) WithReason(reason string) *APIResponse {
	resp.Reason = reason
	return resp
}

// FromBizError 将业务错误转换为API响应
func FromBizError(err *BizError) *APIResponse {
	resp := Failure(int(err.Code), err.Message)
	resp.Reason = err.Reason
	return resp
}

// GRPCStatus 返回 gRPC 状态表示
func (err *BizError) GRPCStatus() *status.Status {
	details := errdetails.ErrorInfo{Reason: err.Reason}
	if err.Metadata != nil {
		details.Metadata = make(map[string]string)
		for k, v := range err.Metadata {
			details.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	base := status.New(ToGRPCCode(err.HTTPStatus()), err.Message)
	withDetails, detailErr := base.WithDetails(&details)
	if detailErr != nil {
		return base
	}
	return withDetails
}

// Code 从错误中提取错误码
func Code(err error) int {
	if err == nil {
		return 0
	}
	if bizErr := new(BizError); errors.As(err, &bizErr) {
		return int(bizErr.Code)
	}
	return int(CodeInternalServer)
}

// Reason 从错误中提取错误原因
func Reason(err error) string {
	if err == nil {
		return ""
	}
	if bizErr := new(BizError); errors.As(err, &bizErr) {
		return bizErr.Reason
	}
	return "InternalError"
}

// FromError 尝试将通用错误转换为业务错误
func FromError(err error) *BizError {
	if err == nil {
		return nil
	}

	// 如果已经是BizError，直接返回
	if bizErr := new(BizError); errors.As(err, &bizErr) {
		return bizErr
	}

	if gs, ok := status.FromError(err); ok {
		return fromGRPCStatus(gs)
	}

	// 未分类错误只能在服务端日志中保留原文，对外统一成稳定的系统错误，避免泄露
	// SQL、连接串、文件路径等内部实现细节。
	return NewBizError(CodeInternalServer, ErrInternal.Reason, ErrInternal.Message)
}

func fromGRPCStatus(gs *status.Status) *BizError {
	bizErr := grpcBizError(gs)
	for _, detail := range gs.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		bizErr.Reason = info.GetReason()
		metadata := make(map[string]any, len(info.GetMetadata()))
		for key, value := range info.GetMetadata() {
			metadata[key] = value
		}
		bizErr.Metadata = metadata
		break
	}
	return bizErr
}

func grpcBizError(gs *status.Status) *BizError {
	switch FromGRPCCode(gs.Code()) {
	case http.StatusBadRequest:
		return NewBizError(CodeUserInvalidCredentials, "Common.InvalidArgument", defaultMessage("Common.BadRequest"))
	case http.StatusNotFound:
		return NewBizError(CodeUserNotFound, "GRPCError", gs.Message())
	case http.StatusUnauthorized:
		return NewBizError(CodeAuthUnauthenticated, ErrUnauthenticated.Reason, ErrUnauthenticated.Message)
	case http.StatusForbidden:
		return NewBizError(CodeUserPermissionDenied, ErrPermissionDenied.Reason, ErrPermissionDenied.Message)
	case http.StatusInternalServerError:
		return NewBizError(CodeInternalServer, ErrInternal.Reason, ErrInternal.Message)
	default:
		return NewBizError(BizCode(11001), "GRPCError", gs.Message())
	}
}

// 协议映射只在初始化时创建，运行期间不修改。
var businessHTTPCodes = map[BizCode]int{
	CodeOK:                     http.StatusOK,
	CodeAuthUnauthenticated:    http.StatusUnauthorized,
	CodeAuthTokenInvalid:       http.StatusUnauthorized,
	CodeAuthTokenExpired:       http.StatusUnauthorized,
	CodeAuthSignToken:          http.StatusInternalServerError,
	CodeUserPermissionDenied:   http.StatusForbidden,
	CodeUserNotFound:           http.StatusNotFound,
	CodeUserAlreadyExists:      http.StatusConflict,
	CodeUserInvalidCredentials: http.StatusBadRequest,
	CodeUserInvalidUsername:    http.StatusBadRequest,
	CodeUserInvalidPassword:    http.StatusBadRequest,
	CodeTooManyRequests:        http.StatusTooManyRequests,
	CodeServiceUnavailable:     http.StatusServiceUnavailable,
	CodeRequestTimeout:         http.StatusGatewayTimeout,
}

var httpGRPCCodes = map[int]codes.Code{
	http.StatusOK:                  codes.OK,
	http.StatusBadRequest:          codes.InvalidArgument,
	http.StatusUnauthorized:        codes.Unauthenticated,
	http.StatusForbidden:           codes.PermissionDenied,
	http.StatusNotFound:            codes.NotFound,
	http.StatusConflict:            codes.AlreadyExists,
	http.StatusTooManyRequests:     codes.ResourceExhausted,
	http.StatusInternalServerError: codes.Internal,
	http.StatusNotImplemented:      codes.Unimplemented,
	http.StatusServiceUnavailable:  codes.Unavailable,
	http.StatusGatewayTimeout:      codes.DeadlineExceeded,
}

var grpcHTTPCodes = map[codes.Code]int{
	codes.OK:                 http.StatusOK,
	codes.Canceled:           http.StatusRequestTimeout,
	codes.Unknown:            http.StatusInternalServerError,
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
	codes.NotFound:           http.StatusNotFound,
	codes.AlreadyExists:      http.StatusConflict,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.FailedPrecondition: http.StatusBadRequest,
	codes.Aborted:            http.StatusConflict,
	codes.OutOfRange:         http.StatusBadRequest,
	codes.Unimplemented:      http.StatusNotImplemented,
	codes.Internal:           http.StatusInternalServerError,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.DataLoss:           http.StatusInternalServerError,
}
