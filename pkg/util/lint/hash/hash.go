package hash

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Doc 记录此检查。
const Doc = `check for correct use of hash.Hash`

// Analyzer 定义此检查。
var Analyzer = &analysis.Analyzer{
	Name:     "hash",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// hashChecker 确保 hash.Hash 接口不被误用。一个常见的错误是假设 Sum 函数返回其输入的哈希，
// 像这样：
//
//	hashedBytes := sha256.New().Sum(inputBytes)
//
// 实际上，Sum 的参数不是要哈希的字节，而是一个将在调用者希望避免分配时用作输出的切片。
// 在上面的示例中，hashedBytes 不是 inputBytes 的 SHA-256 哈希，而是 inputBytes 与空字符串的哈希的连接。
//
// hash.Hash 接口的正确用法如下：
//
//	h := sha256.New()
//	h.Write(inputBytes)
//	hashedBytes := h.Sum(nil)
//
//	h := sha256.New()
//	h.Write(inputBytes)
//	var hashedBytes [sha256.Size]byte
//	h.Sum(hashedBytes[:0])
//
// 为了区分正确和错误的用法，hashChecker 应用一个简单的启发式方法：它标记 a) 参数非 nil 且
// b) 使用返回值的 Sum 调用。
//
// hash.Hash 接口可能会在 Go 2 中得到修复。参见 golang/go#21070。
func run(pass *analysis.Pass) (any, error) {
	stack := make([]ast.Node, 0, 32)
	forAllFiles(pass.Files, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, node)
		if call := hashSumCall(pass, stack); call != nil && !nilArgument(call.Args[0]) && !unusedResult(stack) {
			pass.Reportf(call.Pos(), "probable misuse of hash.Hash.Sum: provide parameter or use return value, but not both")
		}
		return true
	})
	return nil, nil //nolint:nilnil // analysis.Analyzer 没有 ResultType，成功时必须返回 nil 结果。
}

func hashSumCall(pass *analysis.Pass, stack []ast.Node) *ast.CallExpr {
	if len(stack) < 3 {
		return nil
	}
	selector, ok := stack[len(stack)-1].(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Sum" {
		return nil
	}
	value, ok := pass.TypesInfo.Types[selector.X]
	if !ok {
		return nil
	}
	named, ok := value.Type.(*types.Named)
	if !ok || named.Obj().Type().String() != "hash.Hash" {
		return nil
	}
	call, ok := stack[len(stack)-2].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil
	}
	return call
}

func nilArgument(node ast.Node) bool {
	id, ok := node.(*ast.Ident)
	return ok && id.Name == "nil"
}

func unusedResult(stack []ast.Node) bool {
	switch parent := stack[len(stack)-3].(type) {
	case *ast.ExprStmt:
		return true
	case *ast.AssignStmt:
		for i, expression := range parent.Rhs {
			if expression == stack[len(stack)-2] {
				id, ok := parent.Lhs[i].(*ast.Ident)
				return ok && id.Name == "_"
			}
		}
	}
	return false
}

func forAllFiles(files []*ast.File, fn func(node ast.Node) bool) {
	for _, file := range files {
		ast.Inspect(file, fn)
	}
}
