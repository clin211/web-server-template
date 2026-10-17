package otelslog // import "go.opentelemetry.io/contrib/bridges/otelslog"

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// convertValue 将各种类型转换为 attribute.Value。
func convertValue(v any) attribute.Value {
	// 在不使用 reflect 的情况下处理最常见的类型是一个小的性能提升。
	switch val := v.(type) {
	case bool:
		return attribute.BoolValue(val)
	case string:
		return attribute.StringValue(val)
	case time.Duration:
		return attribute.Int64Value(val.Nanoseconds())
	case complex64:
		r := attribute.Float64("r", real(complex128(val)))
		i := attribute.Float64("i", imag(complex128(val)))
		return attribute.MapValue(r, i)
	case complex128:
		r := attribute.Float64("r", real(val))
		i := attribute.Float64("i", imag(val))
		return attribute.MapValue(r, i)
	case time.Time:
		return attribute.Int64Value(val.UnixNano())
	case []byte:
		return attribute.ByteSliceValue(val)
	case error:
		return attribute.StringValue(val.Error())
	case attribute.Value:
		return normalizeAttribute(val)
	}

	return convertOtherValue(v)
}
func convertReflected(v any) attribute.Value {
	t := reflect.TypeOf(v)
	if t == nil {
		return attribute.Value{}
	}
	val := reflect.ValueOf(v)
	switch t.Kind() {
	case reflect.Struct:
		return attribute.StringValue(fmt.Sprintf("%+v", v))
	case reflect.Slice, reflect.Array:
		items := make([]attribute.Value, 0, val.Len())
		for i := range val.Len() {
			items = append(items, convertValue(val.Index(i).Interface()))
		}
		return attribute.SliceValue(items...)
	case reflect.Map:
		return convertMap(val)
	case reflect.Pointer, reflect.Interface:
		if val.IsNil() {
			return attribute.Value{}
		}
		return convertValue(val.Elem().Interface())
	}

	// 尽可能优雅地处理此情况。
	//
	// 不要在这里 panic。让用户提问为什么他们的属性有"unhandled: "前缀
	// 比说他们的代码正在 panic 更可取。
	return attribute.StringValue(fmt.Sprintf("unhandled: (%s) %+v", t, v))
}

// convertUintValue 将 uint64 转换为 attribute.Value。
// 如果值太大而无法放入 int64，它将被转换为字符串。
func convertUintValue(v uint64) attribute.Value {
	if v > math.MaxInt64 {
		return attribute.StringValue(strconv.FormatUint(v, 10))
	}
	return attribute.Int64Value(int64(v))
}

func convertOtherValue(v any) attribute.Value {
	if scalar, ok := convertNumber(v); ok {
		return scalar
	}
	return convertReflected(v)
}
func convertNumber(v any) (attribute.Value, bool) {
	if value, ok := convertSigned(v); ok {
		return value, true
	}
	if value, ok := convertUnsigned(v); ok {
		return value, true
	}
	switch val := v.(type) {
	case float32:
		return attribute.Float64Value(float64(val)), true
	case float64:
		return attribute.Float64Value(val), true
	}
	return attribute.Value{}, false
}
func convertSigned(v any) (attribute.Value, bool) {
	switch val := v.(type) {
	case int:
		return attribute.Int64Value(int64(val)), true
	case int8:
		return attribute.Int64Value(int64(val)), true
	case int16:
		return attribute.Int64Value(int64(val)), true
	case int32:
		return attribute.Int64Value(int64(val)), true
	case int64:
		return attribute.Int64Value(val), true
	}
	return attribute.Value{}, false
}
func convertUnsigned(v any) (attribute.Value, bool) {
	switch val := v.(type) {
	case uint:
		return convertUintValue(uint64(val)), true
	case uint8:
		return attribute.Int64Value(int64(val)), true
	case uint16:
		return attribute.Int64Value(int64(val)), true
	case uint32:
		return attribute.Int64Value(int64(val)), true
	case uint64:
		return convertUintValue(val), true
	case uintptr:
		return convertUintValue(uint64(val)), true
	}
	return attribute.Value{}, false
}

func convertMap(val reflect.Value) attribute.Value {
	kvs := make([]attribute.KeyValue, 0, val.Len())
	for _, k := range val.MapKeys() {
		var key string
		switch k.Kind() {
		case reflect.String:
			key = k.String()
		default:
			key = fmt.Sprintf("%+v", k.Interface())
		}
		kvs = append(kvs, attribute.KeyValue{
			Key:   attribute.Key(key),
			Value: convertValue(val.MapIndex(k).Interface()),
		})
	}
	return attribute.MapValue(kvs...)
}

func normalizeAttribute(v attribute.Value) attribute.Value {
	switch v.Type() {
	case attribute.BOOLSLICE, attribute.INT64SLICE, attribute.FLOAT64SLICE, attribute.STRINGSLICE:
		return convertReflected(v.AsInterface())
	default:
		return v
	}
}
