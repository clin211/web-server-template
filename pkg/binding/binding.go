package binding

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gin-gonic/gin/codec/json"
)

var errInvalidRequest = errors.New("invalid request")

// Bind 按顺序绑定多个来源的数据，并在所有字段就绪后执行一次验证。
func Bind(c *gin.Context, obj any, bindFuncs ...func(*gin.Context, any) error) error {
	for _, bindFunc := range bindFuncs {
		if err := bindFunc(c, obj); err != nil {
			return err
		}
	}
	if binding.Validator != nil {
		return binding.Validator.ValidateStruct(obj)
	}
	return nil
}

// URI 绑定 URI 参数，将验证留给 Bind 统一执行。
func URI(c *gin.Context, obj any) error {
	params := make(map[string][]string, len(c.Params))
	for _, param := range c.Params {
		params[param.Key] = []string{param.Value}
	}
	return binding.MapFormWithTag(obj, params, "uri")
}

// JSON 绑定 JSON 请求体，将验证留给 Bind 统一执行。
func JSON(c *gin.Context, obj any) error {
	if c.Request == nil || c.Request.Body == nil {
		return errInvalidRequest
	}
	decoder := json.API.NewDecoder(c.Request.Body)
	if binding.EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if binding.EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(obj)
}

// Query 绑定 Query 参数，将验证留给 Bind 统一执行。
func Query(c *gin.Context, obj any) error {
	return binding.MapFormWithTag(obj, c.Request.URL.Query(), "form")
}
