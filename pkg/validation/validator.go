package validation

import (
	"fmt"
	"reflect"
)

// ValidatorFunc 定义验证函数类型。
type ValidatorFunc func(value any) error

// Rules 定义字段验证规则。
type Rules map[string]ValidatorFunc

// ValidateAllFields 验证结构体中的所有导出字段。
func ValidateAllFields(obj any, rules Rules) error {
	return ValidateSelectedFields(obj, rules, GetExportedFieldNames(obj)...)
}

// ValidateSelectedFields 验证结构体中的指定导出字段。
func ValidateSelectedFields(obj any, rules Rules, fields ...string) error {
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("%w: got %s", errExpectedStruct, value.Kind())
	}
	for _, field := range fields {
		fieldValue, ok := exportedFieldValue(value, field)
		if !ok {
			continue
		}
		validator, ok := rules[field]
		if !ok {
			continue
		}
		if err := validator(fieldValue.Interface()); err != nil {
			return err
		}
	}
	return nil
}

func exportedFieldValue(value reflect.Value, name string) (reflect.Value, bool) {
	field, ok := value.Type().FieldByName(name)
	if !ok || !field.IsExported() {
		return reflect.Value{}, false
	}
	fieldValue := value.FieldByName(name)
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			return reflect.Value{}, false
		}
		fieldValue = fieldValue.Elem()
	}
	return fieldValue, true
}

// GetExportedFieldNames 返回传入结构体中所有可导出的字段名字。
func GetExportedFieldNames(obj any) []string {
	typ := reflect.TypeOf(obj)
	if typ == nil {
		return []string{}
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return []string{}
	}
	var fields []string
	for field := range typ.Fields() {
		if field.IsExported() {
			fields = append(fields, field.Name)
		}
	}
	return fields
}
