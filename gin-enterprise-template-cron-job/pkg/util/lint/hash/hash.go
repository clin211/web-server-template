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
	selectorIsHash := func(s *ast.SelectorExpr) bool {
		tv, ok := pass.TypesInfo.Types[s.X]
		if !ok {
			return false
		}
		named, ok := tv.Type.(*types.Named)
		if !ok {
			return false
		}
		if named.Obj().Type().String() != "hash.Hash" {
			return false
		}
		return true
	}

	stack := make([]ast.Node, 0, 32)
	forAllFiles(pass.Files, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1] // pop
			return true
		}
		stack = append(stack, n) // 压入栈

		// 查找对 hash.Hash.Sum 的调用。
		selExpr, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if selExpr.Sel.Name != "Sum" {
			return true
		}
		if !selectorIsHash(selExpr) {
			return true
		}
		callExpr, ok := stack[len(stack)-2].(*ast.CallExpr)
		if !ok {
			return true
		}
		if len(callExpr.Args) != 1 {
			return true
		}
		// 我们有一个对 hash.Hash.Sum 的有效调用。

		// 参数是否为 nil？
		var nilArg bool
		if id, ok := callExpr.Args[0].(*ast.Ident); ok && id.Name == "nil" {
			nilArg = true
		}

		// 返回值是否未使用？
		var retUnused bool
	Switch:
		switch t := stack[len(stack)-3].(type) {
		case *ast.AssignStmt:
			for i := range t.Rhs {
				if t.Rhs[i] == stack[len(stack)-2] {
					if id, ok := t.Lhs[i].(*ast.Ident); ok && id.Name == "_" {
						// 赋值给空白标识符不算作使用返回值。
						retUnused = true
					}
					break Switch
				}
			}
			panic("unreachable")
		case *ast.ExprStmt:
			// An expression statement means the return value is unused.
			retUnused = true
		default:
		}

		if !nilArg && !retUnused {
			pass.Reportf(callExpr.Pos(), "probable misuse of hash.Hash.Sum: "+
				"提供参数或使用返回值，但不能同时使用两者")
		}
		return true
	})

	return nil, nil
}

func forAllFiles(files []*ast.File, fn func(node ast.Node) bool) {
	for _, f := range files {
		ast.Inspect(f, fn)
	}
}
