package pass

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

// FindContainingFile 在 Pass 中查找 ast 节点的文件。
func FindContainingFile(pass *analysis.Pass, n ast.Node) *ast.File {
	fPos := pass.Fset.File(n.Pos())
	for _, f := range pass.Files {
		if pass.Fset.File(f.Pos()) == fPos {
			return f
		}
	}
	panic(fmt.Errorf("cannot file file for %v", n))
}

// HasNolintComment 如果传递的节点上的注释包含注释 "nolint:<nolintName>"，则返回 true。
func HasNolintComment(pass *analysis.Pass, n ast.Node, nolintName string) bool {
	f := FindContainingFile(pass, n)
	relevant, containing := findNodesInBlock(f, n)
	cm := ast.NewCommentMap(pass.Fset, containing, f.Comments)
	// 检查任何相关的 ast 节点是否包含相关注释。
	nolintComment := "nolint:" + nolintName
	for _, cn := range relevant {
		// Ident 节点在 containing 中包含 ident 的所有注释。这可能太多了。
		// 想象一下，在声明块的其他地方的 ident 上有一个注释，我们不希望它影响以后的转换。
		//
		// 我们希望将其过滤到相关语句内 ident 上的所有注释。
		// 为此，我们拒绝 slash 在最外层相关节点之外的 ident 上的注释。
		// 我们通过检查注释相对于最外层相关节点的位置来做到这一点。
		// 这是合理的，因为我们不能单独在 ident 上有注释，因为 ident 不是语句。
		_, isIdent := cn.(*ast.Ident)
		for _, cg := range cm[cn] {
			for _, c := range cg.List {
				if !strings.Contains(c.Text, nolintComment) {
					continue
				}
				outermost := relevant[len(relevant)-1]
				if isIdent && (cg.Pos() < outermost.Pos() || cg.End() > outermost.End()) {
					continue
				}
				return true
			}
		}
	}
	return false
}

// findNodesInBlock 查找最接近 n 的块或声明下的所有表达式和语句。
// 这个想法是我们要找到包含 ast 节点 n 的块或声明中出现的注释，用于过滤。
// 我们希望将注释过滤到与 n 或最接近的封闭块或声明中的语句之前的任何表达式相关联的所有注释。
// 这是为了处理多行表达式或相关表达式出现在 if 或 for 的 init 子句中而注释在前一行的情况。
//
// 例如，假设 n 是以下片段中方法 foo 上的 *ast.CallExpr：
//
//	func nonsense() bool {
//	    if v := (g.foo() + 1) > 2; !v {
//	        return true
//	    }
//	    return false
//	}
//
// 此函数将返回直到 `IfStmt` 的所有节点作为相关节点，
// 并将函数 nonsense 的 `BlockStmt` 作为包含节点。
func findNodesInBlock(f *ast.File, n ast.Node) (relevant []ast.Node, containing ast.Node) {
	stack, _ := astutil.PathEnclosingInterval(f, n.Pos(), n.End())
	// 将 n 的所有子节点添加到相关节点集合。
	ast.Walk(funcVisitor(func(node ast.Node) {
		relevant = append(relevant, node)
	}), n)

	// 节点刚刚被添加，父节点在开头，子节点在结尾。反转它。
	reverseNodes(relevant)

	// 添加父节点直到封闭块或声明块并发现包含节点。
	containing = f // 最坏情况
	for _, n := range stack {
		switch n.(type) {
		case *ast.GenDecl, *ast.BlockStmt:
			containing = n
			return relevant, containing
		default:
			// 将 n 的所有父节点直到包含 BlockStmt 或 GenDecl 添加到相关节点集合。
			relevant = append(relevant, n)
		}
	}
	return relevant, containing
}

func reverseNodes(n []ast.Node) {
	for i := 0; i < len(n)/2; i++ {
		n[i], n[len(n)-i-1] = n[len(n)-i-1], n[i]
	}
}

type funcVisitor func(node ast.Node)

var _ ast.Visitor = (funcVisitor)(nil)

func (f funcVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node != nil {
		f(node)
	}
	return f
}
