package errorsx

import (
	"errors"
)

// Is 报告 err 链中的任何错误是否匹配 target。
//
// 该链由 err 本身以及通过重复调用 Unwrap 获得的错误序列组成。
//
// 如果错误等于目标错误，或者错误实现了 Is(error) bool 方法且
// Is(target) 返回 true，则认为该错误匹配目标。
func Is(err, target error) bool { return errors.Is(err, target) }

// As 在 err 的链中查找第一个匹配 target 的错误，如果找到，将 target 设置为该错误值并返回 true。
//
// 该链由 err 本身以及通过重复调用 Unwrap 获得的错误序列组成。
//
// 如果错误的具体值可赋值给 target 指向的值，或者错误具有 As(interface{}) bool 方法
// 且 As(target) 返回 true，则错误匹配 target。在后一种情况下，As 方法负责设置 target。
//
// 如果 target 不是指向实现 error 的类型的非空指针，或不是指向任何接口类型的指针，As 将 panic。
// 如果 err 为 nil，As 返回 false。
func As(err error, target interface{}) bool { return errors.As(err, target) }

// Unwrap 返回在 err 上调用 Unwrap 方法的结果，如果 err 的类型包含返回 error 的 Unwrap 方法。
// 否则，Unwrap 返回 nil。
func Unwrap(err error) error {
	return errors.Unwrap(err)
}
