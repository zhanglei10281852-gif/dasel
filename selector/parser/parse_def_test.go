package parser_test

import (
	"strings"
	"testing"

	"github.com/tomwright/dasel/v3/selector/ast"
	"github.com/tomwright/dasel/v3/selector/lexer"
	"github.com/tomwright/dasel/v3/selector/parser"
)

func parse(t *testing.T, input string) (ast.Expr, error) {
	t.Helper()
	tokens, err := lexer.NewTokenizer(input).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	return parser.NewParser(tokens).Parse()
}

func TestParser_Parse_Def(t *testing.T) {
	t.Run("single function declaration", func(t *testing.T) {
		got, err := parse(t, `def label($suffix) { name + $suffix }; items.map(label("!"))`)
		if err != nil {
			t.Fatal(err)
		}

		program, ok := got.(ast.ProgramExpr)
		if !ok {
			t.Fatalf("expected ProgramExpr, got %T", got)
		}
		if len(program.Functions) != 1 {
			t.Fatalf("expected 1 function, got %d", len(program.Functions))
		}

		fn := program.Functions[0]
		if fn.Name != "label" {
			t.Errorf("unexpected function name: %s", fn.Name)
		}
		if len(fn.Params) != 1 || fn.Params[0] != "suffix" {
			t.Errorf("unexpected params: %v", fn.Params)
		}
		body, ok := fn.Body.(ast.BinaryExpr)
		if !ok {
			t.Fatalf("expected BinaryExpr body, got %T", fn.Body)
		}
		if _, ok := body.Left.(ast.PropertyExpr); !ok {
			t.Errorf("expected PropertyExpr body left side, got %T", body.Left)
		}
		if rv, ok := body.Right.(ast.VariableExpr); !ok || rv.Name != "suffix" {
			t.Errorf("unexpected body right side: %T", body.Right)
		}

		chain, ok := program.Expr.(ast.ChainedExpr)
		if !ok {
			t.Fatalf("expected ChainedExpr main query, got %T", program.Expr)
		}
		if len(chain.Exprs) != 2 {
			t.Fatalf("expected 2 chained expressions, got %d", len(chain.Exprs))
		}
		mapExpr, ok := chain.Exprs[1].(ast.MapExpr)
		if !ok {
			t.Fatalf("expected MapExpr, got %T", chain.Exprs[1])
		}
		call, ok := mapExpr.Expr.(ast.CallExpr)
		if !ok {
			t.Fatalf("expected CallExpr map body, got %T", mapExpr.Expr)
		}
		if call.Function != "label" || len(call.Args) != 1 {
			t.Errorf("unexpected call expression: %+v", call)
		}
	})

	t.Run("multiple declarations with several params", func(t *testing.T) {
		got, err := parse(t, `def a() { 1 }; def b($x, $y) { $x + $y }; b(1, 2)`)
		if err != nil {
			t.Fatal(err)
		}
		program, ok := got.(ast.ProgramExpr)
		if !ok {
			t.Fatalf("expected ProgramExpr, got %T", got)
		}
		if len(program.Functions) != 2 {
			t.Fatalf("expected 2 functions, got %d", len(program.Functions))
		}
		if program.Functions[0].Name != "a" || len(program.Functions[0].Params) != 0 {
			t.Errorf("unexpected first function: %+v", program.Functions[0])
		}
		if program.Functions[1].Name != "b" {
			t.Errorf("unexpected second function name: %s", program.Functions[1].Name)
		}
		if len(program.Functions[1].Params) != 2 ||
			program.Functions[1].Params[0] != "x" || program.Functions[1].Params[1] != "y" {
			t.Errorf("unexpected second function params: %v", program.Functions[1].Params)
		}
	})

	t.Run("body supports semicolon separated statements", func(t *testing.T) {
		got, err := parse(t, `def f($x) { $y = $x + 1; $y }; f(1)`)
		if err != nil {
			t.Fatal(err)
		}
		program := got.(ast.ProgramExpr)
		body, ok := program.Functions[0].Body.(ast.ChainedExpr)
		if !ok {
			t.Fatalf("expected ChainedExpr body, got %T", program.Functions[0].Body)
		}
		if len(body.Exprs) != 2 {
			t.Fatalf("expected 2 body statements, got %d", len(body.Exprs))
		}
	})

	t.Run("no declarations returns plain expression", func(t *testing.T) {
		got, err := parse(t, `foo.bar`)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := got.(ast.ProgramExpr); ok {
			t.Fatal("query without declarations must not be wrapped in a ProgramExpr")
		}
	})

	t.Run("def remains usable as a data field", func(t *testing.T) {
		got, err := parse(t, `def`)
		if err != nil {
			t.Fatal(err)
		}
		prop, ok := got.(ast.PropertyExpr)
		if !ok {
			t.Fatalf("expected PropertyExpr, got %T", got)
		}
		if s, ok := prop.Property.(ast.StringExpr); !ok || s.Value != "def" {
			t.Errorf("unexpected property: %+v", prop)
		}
	})

	t.Run("def followed by non-call stays a property", func(t *testing.T) {
		got, err := parse(t, `def.name`)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := got.(ast.ProgramExpr); ok {
			t.Fatal("def without the name( shape must not start a declaration")
		}
	})

	t.Run("declaration errors", func(t *testing.T) {
		cases := []struct {
			name    string
			input   string
			wantErr string
		}{
			{
				name:    "duplicate function",
				input:   `def f() { 1 }; def f() { 2 }; f()`,
				wantErr: "defined more than once",
			},
			{
				name:    "duplicate parameter",
				input:   `def f($x, $x) { $x }; f(1)`,
				wantErr: "duplicate parameter",
			},
			{
				name:    "non variable parameter",
				input:   `def f(x) { x }; f(1)`,
				wantErr: "variables prefixed with $",
			},
			{
				name:    "missing body closing brace",
				input:   `def f() { 1; f()`,
				wantErr: "unexpected token",
			},
			{
				name:    "missing semicolon",
				input:   `def f() { 1 } f()`,
				wantErr: "must end with a semicolon",
			},
			{
				name:    "empty body",
				input:   `def f() {}; f()`,
				wantErr: "body must contain at least one expression",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := parse(t, tc.input)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %s", tc.wantErr, err)
				}
			})
		}
	})
}
