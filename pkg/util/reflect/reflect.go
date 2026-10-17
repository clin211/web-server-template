package reflect

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	errUnknownField       = errors.New("unknown field")
	errUndefinedGORMField = errors.New("undefined gorm field")
)

func ToGormDBMap(obj any, fields []string) (map[string]any, error) {
	reflectType := reflect.ValueOf(obj).Type()
	reflectValue := reflect.ValueOf(obj)
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Pointer {
		reflectType = reflectType.Elem()
		reflectValue = reflect.ValueOf(obj).Elem()
	}

	ret := make(map[string]any, 0)
	for _, f := range fields {
		fs, exist := reflectType.FieldByName(f)
		if !exist {
			return nil, fmt.Errorf("%w: %s", errUnknownField, f)
		}

		tagMap := parseTagSetting(fs.Tag)
		gormfiled, exist := tagMap["COLUMN"]
		if !exist {
			return nil, fmt.Errorf("%w: %s", errUndefinedGORMField, f)
		}

		ret[gormfiled] = reflectValue.FieldByName(f)
	}
	return ret, nil
}

func parseTagSetting(tags reflect.StructTag) map[string]string {
	setting := map[string]string{}
	for _, str := range []string{tags.Get("sql"), tags.Get("gorm")} {
		if str == "" {
			continue
		}
		tags := strings.SplitSeq(str, ";")
		for value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if len(v) >= 2 {
				setting[k] = strings.Join(v[1:], ":")
			} else {
				setting[k] = k
			}
		}
	}
	return setting
}

func GetObjFieldsMap(obj any, fields []string) map[string]any {
	ret := make(map[string]any)

	modelReflect := reflect.ValueOf(obj)
	if modelReflect.Kind() == reflect.Pointer {
		modelReflect = modelReflect.Elem()
	}

	modelRefType := modelReflect.Type()
	fieldsCount := modelReflect.NumField()
	var fieldData any
	for i := range fieldsCount {
		field := modelReflect.Field(i)
		if len(fields) != 0 && !findString(fields, modelRefType.Field(i).Name) {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			fallthrough
		case reflect.Pointer:
			fieldData = GetObjFieldsMap(field.Interface(), []string{})
		default:
			fieldData = field.Interface()
		}

		ret[modelRefType.Field(i).Name] = fieldData
	}

	return ret
}

func CopyObj(from any, to any, fields []string) (bool, error) {
	fromMap := GetObjFieldsMap(from, fields)
	toMap := GetObjFieldsMap(to, fields)
	if reflect.DeepEqual(fromMap, toMap) {
		return false, nil
	}

	t := reflect.ValueOf(to).Elem()
	for k, v := range fromMap {
		val := t.FieldByName(k)
		val.Set(reflect.ValueOf(v))
	}
	return true, nil
}

// CopyObjViaYaml 将 "from" 序列化为 yaml 数据，然后将数据反序列化到 "to"。
func CopyObjViaYaml(to any, from any) error {
	if from == nil || to == nil {
		return nil
	}

	data, err := yaml.Marshal(from)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, to)
}

// StructName 用于从对象获取结构体名称。
func StructName(obj any) string {
	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Pointer {
		return t.Elem().Name()
	}

	return t.Name()
}

// findString 如果目标在切片中返回 true，否则返回 false。
func findString(slice []string, target string) bool {
	return slices.Contains(slice, target)
}
