package localization

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"text/template"
	"text/template/parse"
)

func TestCatalogTemplateParametersMatch(t *testing.T) {
	catalogs := make([]map[string]string, 0, 2)
	for _, name := range []string{"locales/en.json", "locales/zh-Hans.json"} {
		data, err := catalogFS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		catalogs = append(catalogs, catalog)
	}
	for id, source := range catalogs[0] {
		left := templateFields(t, id+"/en", source)
		right := templateFields(t, id+"/zh-Hans", catalogs[1][id])
		if !maps.Equal(left, right) {
			t.Errorf("%s: English fields %v differ from Chinese fields %v", id, left, right)
		}
	}
}

func templateFields(t *testing.T, name, source string) map[string]struct{} {
	t.Helper()
	tmpl, err := template.New(name).Parse(source)
	if err != nil {
		t.Fatalf("invalid template %s: %v", name, err)
	}
	fields := make(map[string]struct{})
	var walk func(parse.Node)
	walk = func(node parse.Node) {
		switch n := node.(type) {
		case *parse.ListNode:
			if n != nil {
				for _, child := range n.Nodes {
					walk(child)
				}
			}
		case *parse.ActionNode:
			walk(n.Pipe)
		case *parse.PipeNode:
			if n != nil {
				for i, command := range n.Cmds {
					collectIndexFields(t, name, n, i, fields)
					walk(command)
				}
			}
		case *parse.CommandNode:
			for _, arg := range n.Args {
				walk(arg)
			}
		case *parse.FieldNode:
			fields[strings.Join(n.Ident, ".")] = struct{}{}
		case *parse.ChainNode:
			fields[strings.Join(n.Field, ".")] = struct{}{}
			walk(n.Node)
		case *parse.VariableNode:
			if len(n.Ident) > 1 {
				fields[strings.Join(n.Ident[1:], ".")] = struct{}{}
			}
		case *parse.IfNode:
			walk(n.Pipe)
			walk(n.List)
			walk(n.ElseList)
		case *parse.RangeNode:
			walk(n.Pipe)
			walk(n.List)
			walk(n.ElseList)
		case *parse.WithNode:
			walk(n.Pipe)
			walk(n.List)
			walk(n.ElseList)
		case *parse.TemplateNode:
			walk(n.Pipe)
		}
	}
	for _, defined := range tmpl.Templates() {
		walk(defined.Tree.Root)
	}
	return fields
}

func collectIndexFields(t *testing.T, name string, pipe *parse.PipeNode, position int, fields map[string]struct{}) {
	t.Helper()
	command := pipe.Cmds[position]
	if len(command.Args) < 2 {
		return
	}
	function, ok := command.Args[0].(*parse.IdentifierNode)
	if !ok || function.Ident != "index" {
		return
	}
	for _, arg := range command.Args[2:] {
		collectLiteralIndexField(t, name, arg, fields)
	}
	if position > 0 {
		// Go 管道将上一条命令的输出作为 index 的最后一个参数。
		previous := pipe.Cmds[position-1]
		if len(previous.Args) != 1 {
			t.Fatalf("%s: catalog index keys must be literals for parameter parity checks", name)
		}
		collectLiteralIndexField(t, name, previous.Args[0], fields)
	}
}

func collectLiteralIndexField(t *testing.T, name string, node parse.Node, fields map[string]struct{}) {
	t.Helper()
	switch key := node.(type) {
	case *parse.StringNode:
		fields[key.Text] = struct{}{}
		return
	case *parse.NumberNode:
		// 字符串的数值索引不是参数名。
		return
	case *parse.PipeNode:
		if len(key.Cmds) == 1 && len(key.Cmds[0].Args) == 1 {
			collectLiteralIndexField(t, name, key.Cmds[0].Args[0], fields)
			return
		}
	}
	t.Fatalf("%s: catalog index keys must be literals for parameter parity checks", name)
}

func TestTemplateParameterAccessors(t *testing.T) {
	for _, source := range []string{
		`Limit {{.MaxLength}}`,
		`Limit {{index . "MaxLength"}}`,
		`Limit {{"MaxLength" | index .}}`,
		`Limit {{(("MaxLength")) | index .}}`,
		`Limit {{(.).MaxLength}}`,
		`Limit {{(.).MaxLength | printf "%d"}}`,
		`{{$root := .}}Limit {{$root.MaxLength}}`,
		`{{$root := .}}Limit {{($root).MaxLength}}`,
		`{{$root := .}}Limit {{"MaxLength" | index $root}}`,
		`{{$root := .}}Limit {{index $root "MaxLength"}}`,
		`{{$limit := .MaxLength}}Limit {{$limit}}`,
		`{{define "limit"}}Limit {{.MaxLength}}{{end}}{{template "limit" .}}`,
	} {
		got := templateFields(t, "accessor", source)
		if !maps.Equal(got, map[string]struct{}{"MaxLength": {}}) {
			t.Errorf("parameters in %q = %v, want MaxLength", source, got)
		}
	}
}
