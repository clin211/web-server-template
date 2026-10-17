package validation

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/klog/v2"
)

var (
	errExpectedStruct = errors.New("expected a struct or a pointer to a struct")
	errMissingField   = errors.New("field does not exist in struct")
	errRequiredField  = errors.New("field must be provided")
)

// Validator 实现了 validate.IValidator 接口。
type Validator struct {
	registry map[string]reflect.Value
}

// ProviderSet 是验证器的提供者。
//var ProviderSet = wire.NewSet(NewValidator)

// NewValidator 创建并初始化一个自定义验证器。
func NewValidator(customValidator any) *Validator {
	return &Validator{registry: extractValidationMethods(customValidator)}
}

// Validate 使用适当的验证方法验证请求。
func (v *Validator) Validate(ctx context.Context, request any) error {
	validationFunc, ok := v.registry[reflect.TypeOf(request).Elem().Name()]
	if !ok {
		return nil // 未找到该请求类型的验证函数
	}

	result := validationFunc.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(request)})
	if !result[0].IsNil() {
		err, ok := result[0].Interface().(error)
		if !ok {
			return fmt.Errorf("validation result: %w", errExpectedStruct)
		}
		return err
	}

	return nil
}

// extractValidationMethods 从提供的自定义验证器中提取并返回验证函数映射。
func extractValidationMethods(customValidator any) map[string]reflect.Value {
	funcs := make(map[string]reflect.Value)
	validatorType := reflect.TypeOf(customValidator)
	validatorValue := reflect.ValueOf(customValidator)

	for method := range validatorType.Methods() {
		methodValue := validatorValue.MethodByName(method.Name)

		if !isValidationMethod(method.Name, methodValue) {
			continue
		}
		methodType := methodValue.Type()

		// 确保方法名称符合预期的命名约定
		requestTypeName := methodType.In(1).Elem().Name()
		if method.Name != ("Validate" + requestTypeName) {
			continue
		}

		klog.V(4).InfoS("Registering validator", "validator", requestTypeName)
		funcs[requestTypeName] = methodValue
	}

	return funcs
}

// ValidRequired 验证结构体中的必需字段是否存在且不为空.
func ValidRequired(obj any, requiredFields ...string) error {
	val := reflect.ValueOf(obj)

	// 检查 obj 是否为结构体或指向结构体的指针
	if val.Kind() != reflect.Struct && val.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: got %s", errExpectedStruct, val.Kind())
	}

	// 如果是指针，获取指向的结构体值
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	// 仍需确保最终值是结构体
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("%w: got %s", errExpectedStruct, val.Kind())
	}

	// 遍历需要验证的字段
	for _, field := range requiredFields {
		// 使用反射获取字段
		fieldVal := val.FieldByName(field)

		// 判断字段是否存在
		if !fieldVal.IsValid() {
			return fmt.Errorf("%w: %s", errMissingField, field)
		}

		// 检查字段是否为 nil
		if fieldVal.IsNil() {
			return fmt.Errorf("%w: %s", errRequiredField, field)
		}
	}

	return nil
}

func isValidationMethod(name string, value reflect.Value) bool {
	if !value.IsValid() || !strings.HasPrefix(name, "Validate") {
		return false
	}
	typ := value.Type()
	return typ.NumIn() == 2 && typ.NumOut() == 1 &&
		typ.In(0) == reflect.TypeFor[context.Context]() && typ.In(1).Kind() == reflect.Pointer &&
		typ.Out(0) == reflect.TypeFor[error]()
}
