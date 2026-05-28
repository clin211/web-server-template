---
name: Effective Go
description: "应用 Go 最佳实践、惯用语和惯例，参考 golang.org/doc/effective_go。在编写、审查或重构 Go 代码时使用，以确保实现符合 Go 风格、简洁高效。"
---

# Effective Go

应用官方 [Effective Go 指南](https://go.dev/doc/effective_go) 中的最佳实践和惯例，编写符合 Go 风格、简洁优雅的代码。

## 适用场景

在以下情况下自动应用本技能：

- 编写新的 Go 代码
- 审查 Go 代码
- 重构现有的 Go 实现

## 核心要点

遵循 <https://go.dev/doc/effective_go> 中记录的约定和模式，特别关注：

- **代码格式**：务必使用 `gofmt` - 这是不可妥协的要求
- **命名规范**：不使用下划线，导出名称使用 MixedCaps，未导出名称使用 mixedCaps
- **错误处理**：始终检查错误；返回错误，不要 panic
- **并发编程**：通过通信来共享内存（使用 channel）
- **接口设计**：保持接口小巧（1-3 个方法为佳）；接收接口参数，返回具体类型
- **文档注释**：为所有导出符号编写文档，以符号名称开头

## 参考资料

- 官方指南：<https://go.dev/doc/effective_go>
- 代码审查评论：<https://github.com/golang/go/wiki/CodeReviewComments>
- 标准库：作为惯用模式的参考
