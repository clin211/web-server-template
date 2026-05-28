package ptr

import (
	"fmt"
	"reflect"
)

// AllPtrFieldsNil 检查结构体中所有指针字段是否都为 nil。这在以下场景很有用：
// 例如，当一个 API 结构体由插件处理时，插件需要区分
// "没有插件接受此规范"和"此规范为空"的情况。
//
// 此函数仅对结构体和指向结构体的指针有效。任何其他类型
// 都会导致 panic。传入类型化的 nil 指针将返回 true。
func AllPtrFieldsNil(obj interface{}) bool {
	v := reflect.ValueOf(obj)
	if !v.IsValid() {
		panic(fmt.Sprintf("reflect.ValueOf() produced a non-valid Value for %#v", obj))
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() == reflect.Ptr && !v.Field(i).IsNil() {
			return false
		}
	}
	return true
}

// To 返回指向给定值的指针。
func To[T any](v T) *T {
	return &v
}

// From 返回指针 p 指向的值。
// 如果指针为 nil，则返回 T 的零值。
func From[T any](v *T) T {
	var zero T
	if v != nil {
		return *v
	}

	return zero
}

// FromOr 解引用 ptr 并返回其指向的值（如果不为 nil），否则
// 返回 def。
func FromOr[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}

// IsNil 返回给定指针 v 是否为 nil。
func IsNil[T any](p *T) bool {
	return p == nil
}

// IsNotNil 是 [IsNil] 的否定形式。
func IsNotNil[T any](p *T) bool {
	return p != nil
}

// Clone 返回该值的浅拷贝。
// 如果给定的指针为 nil，则返回 nil。
//
// 提示：元素是通过赋值（=）复制的，因此这是浅拷贝。
// 如果要进行深拷贝，请使用 [CloneBy] 并传入适当的元素
// 克隆函数。
//
// 别名：Copy
func Clone[T any](p *T) *T {
	if p == nil {
		return nil
	}
	clone := *p
	return &clone
}

// CloneBy 是 [Clone] 的变体，它返回该值的拷贝。
// 元素使用函数 f 进行复制。
// 如果给定的指针为 nil，则返回 nil。
func CloneBy[T any](p *T, f func(T) T) *T {
	return Map(p, f)
}

// Equal 如果两个参数都为 nil 或两个参数解引用后
// 的值相等，则返回 true。
func Equal[T comparable](a, b *T) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return *a == *b
}

// EqualTo 返回指针 p 的值是否等于值 v。
// 这是 "x != nil && *x == y" 的简写形式。
//
// 示例：
//
//	x, y := 1, 2
//	Equal(&x, 1)   ⏩  true
//	Equal(&y, 1)   ⏩  false
//	Equal(nil, 1)  ⏩  false
func EqualTo[T comparable](p *T, v T) bool {
	return p != nil && *p == v
}

// Map 将函数 f 应用于指针 p 的元素。
// 如果 p 为 nil，则不会调用 f 并返回 nil，否则，
// 将 f 的结果作为新指针返回。
//
// 示例：
//
//	i := 1
//	Map(&i, strconv.Itoa)       ⏩  (*string)("1")
//	Map[int](nil, strconv.Itoa) ⏩  (*string)(nil)
func Map[F, T any](p *F, f func(F) T) *T {
	if p == nil {
		return nil
	}
	return To(f(*p))
}
