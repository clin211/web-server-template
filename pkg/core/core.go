package core

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/clin211/gin-enterprise-template/pkg/binding"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

// Validator 是验证函数的类型，用于对绑定的数据结构进行验证.
type Validator[T any] func(context.Context, *T) error

// Binder 定义绑定函数的类型，用于绑定请求数据到相应结构体.
type Binder func(any) error

// Handler 是处理函数的类型，用于处理已经绑定和验证的数据.
type Handler[T any, R any] func(ctx context.Context, req *T) (R, error)

// successMessage 是所有成功响应统一使用的 message 文案。
const successMessage = "success"

// HandleAllRequest 是处理综合请求的快捷函数。
// 它会将 URI 参数、Query/Form 参数、以及 JSON Body 一并绑定到结构体中，
// 然后可选执行多个验证器，最后调用业务逻辑处理函数。
func HandleAllRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	var request T

	// 绑定和验证请求数据
	if err := ShouldBindAll(c, &request, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}

	// 调用实际的业务逻辑处理函数
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

// HandleJSONRequest 是处理 JSON 请求的快捷函数.
func HandleJSONRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	HandleRequest(c, c.ShouldBindJSON, handler, validators...)
}

// HandleUriJSONRequest 是处理同时包含 URI 参数和 JSON 请求体的快捷函数。
func HandleUriJSONRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	var request T
	if err := ShouldBindUriJSON(c, &request, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

// HandleNoBodyRequest 是处理无请求体的快捷函数.
// 用于不需要请求 body 的接口（如刷新令牌等），直接创建空的请求结构体并调用处理函数.
func HandleNoBodyRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	var request T

	// 跳过绑定，直接执行验证和业务逻辑
	if err := FinalizeRequest(c, &request, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}

	// 调用实际的业务逻辑处理函数
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

// HandleQueryRequest 是处理 Query 参数请求的快捷函数.
func HandleQueryRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	HandleRequest(c, c.ShouldBindQuery, handler, validators...)
}

func HandleUriQueryRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	var request T
	if err := ShouldBindUriQuery(c, &request, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

func HandleProtoJSONRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	var request T
	if err := ShouldBindProtoJSON(c, &request, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

func HandleUriProtoJSONRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	var request T
	if err := ShouldBindUriProtoJSON(c, &request, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

// HandleUriRequest 是处理 URI 请求的快捷函数.
func HandleUriRequest[T any, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T]) {
	HandleRequest(c, c.ShouldBindUri, handler, validators...)
}

// HandleRequest 是通用的请求处理函数.
// 负责绑定请求数据、执行验证、并调用实际的业务处理逻辑函数.
func HandleRequest[T any, R any](c *gin.Context, binder Binder, handler Handler[T, R], validators ...Validator[T]) {
	var request T

	// 绑定和验证请求数据
	if err := ReadRequest(c, &request, binder, validators...); err != nil {
		WriteResponse(c, nil, err)
		return
	}

	// 调用实际的业务逻辑处理函数
	response, err := handler(c.Request.Context(), &request)
	WriteResponse(c, response, err)
}

// ShouldBindJSON 使用 JSON 格式的绑定函数绑定请求参数并执行验证。
func ShouldBindJSON[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	return ReadRequest(c, rq, c.ShouldBindJSON, validators...)
}

// ShouldBindQuery 使用 Query 格式的绑定函数绑定请求参数并执行验证。
func ShouldBindQuery[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	return ReadRequest(c, rq, c.ShouldBindQuery, validators...)
}

// ShouldBindUri 使用 URI 格式的绑定函数绑定请求参数并执行验证。
func ShouldBindUri[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	return ReadRequest(c, rq, c.ShouldBindUri, validators...)
}

func ShouldBindUriJSON[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	if err := binding.Bind(c, rq, binding.JSON, binding.URI); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}
	return FinalizeRequest(c, rq, validators...)
}

func ShouldBindUriQuery[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	if err := binding.Bind(c, rq, binding.URI, binding.Query); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}
	return FinalizeRequest(c, rq, validators...)
}

func ShouldBindProtoJSON[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	if err := bindProtoJSON(c, rq); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}
	return FinalizeRequest(c, rq, validators...)
}

func ShouldBindUriProtoJSON[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	if err := bindProtoJSON(c, rq); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}
	if err := binding.URI(c, rq); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}
	return FinalizeRequest(c, rq, validators...)
}

func bindProtoJSON[T any](c *gin.Context, rq *T) error {
	message, ok := any(rq).(proto.Message)
	if !ok {
		return errorsx.ErrBind.WithMessage("request is not a protobuf message")
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(body, message)
}

// ShouldBindAll 将 URI 参数、Query/Form 参数以及 JSON Body 一并绑定到结构体中。
// 它能覆盖同名字段（后者优先），并支持 Default() 与验证函数 validators。
func ShouldBindAll[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	if err := binding.Bind(c, rq, binding.URI, binding.JSON); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}

	// 应用 Default() 并执行验证逻辑
	if err := FinalizeRequest(c, rq, validators...); err != nil {
		return err
	}

	return nil
}

// ReadRequest 是用于绑定和验证请求数据的通用工具函数.
// - 它负责调用绑定函数绑定请求数据.
// - 如果目标类型实现了 Default 接口，会调用其 Default 方法设置默认值.
// - 最后执行传入的验证器对数据进行校验.
func ReadRequest[T any](c *gin.Context, rq *T, binder Binder, validators ...Validator[T]) error {
	// 调用绑定函数绑定请求数据
	if err := binder(rq); err != nil {
		return errorsx.ErrBind.WithDetails(err.Error())
	}

	if err := FinalizeRequest(c, rq, validators...); err != nil {
		return err
	}

	return nil
}

// FinalizeRequest 在请求参数绑定完成后执行以下操作：
// 1. 如果目标类型实现了 Default() 方法，则调用 Default 设置默认值；
// 2. 顺序执行所有验证函数。
func FinalizeRequest[T any](c *gin.Context, rq *T, validators ...Validator[T]) error {
	// 应用默认值
	if defaulter, ok := any(rq).(interface{ Default() }); ok {
		defaulter.Default()
	}

	// 执行验证逻辑
	for _, validate := range validators {
		if validate == nil {
			continue
		}
		if err := validate(c.Request.Context(), rq); err != nil {
			return err
		}
	}

	return nil
}

// WriteResponse 是通用的响应函数.
// 它会根据是否发生错误，生成成功响应或标准化的错误响应.
func WriteResponse(c *gin.Context, data any, err error) {
	if err != nil {
		// 如果发生错误，生成错误响应
		bizErr := errorsx.FromError(err)
		response := errorsx.FromBizError(bizErr)
		c.JSON(bizErr.HTTPStatus(), response)
		return
	}

	// proto.Message 走 protojson（EmitUnpopulated + 枚举名），与 OpenAPI 契约
	// 中 XxxResponseEnvelope 的字段声明保持一致；注意必须传入指针，值类型无法
	// 通过断言，会退化到 encoding/json 并丢失零值字段。
	if message, ok := data.(proto.Message); ok {
		payload, marshalErr := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(message)
		if marshalErr != nil {
			// 序列化失败属于程序错误：返回 500 内部错误，而不是输出一个
			// 与契约不一致的"部分成功"响应。
			slog.ErrorContext(c.Request.Context(), "Failed to marshal proto response", "error", marshalErr)
			c.JSON(http.StatusInternalServerError, errorsx.FromBizError(errorsx.ErrInternal))
			return
		}

		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte("{\"code\":0,\"message\":\""+successMessage+"\",\"data\":"+string(payload)+"}"))
		return
	}

	c.JSON(http.StatusOK, errorsx.Success(data, successMessage))
}

// WriteBizError 写入业务错误的便捷函数
func WriteBizError(c *gin.Context, bizErr *errorsx.BizError) {
	response := errorsx.FromBizError(bizErr)
	c.JSON(bizErr.HTTPStatus(), response)
}
