package ast

import "testing"

// TestExpr_expr tests the expr method of all the types in the ast package.
// Note that this doesn't actually do anything and is just forcing test coverage.
// The expr func only exists for type safety with the Expr interface.
func TestExpr_expr(t *testing.T) {
	NumberFloatExpr{}.expr()
	NumberIntExpr{}.expr()
	StringExpr{}.expr()
	BoolExpr{}.expr()
	BinaryExpr{}.expr()
	UnaryExpr{}.expr()
	CallExpr{}.expr()
	ChainedExpr{}.expr()
	SpreadExpr{}.expr()
	RangeExpr{}.expr()
	IndexExpr{}.expr()
	ArrayExpr{}.expr()
	PropertyExpr{}.expr()
	ObjectExpr{}.expr()
	MapExpr{}.expr()
	EachExpr{}.expr()
	VariableExpr{}.expr()
	GroupExpr{}.expr()
	ConditionalExpr{}.expr()
	BranchExpr{}.expr()
	FilterExpr{}.expr()
	SearchExpr{}.expr()
	RecursiveDescentExpr{}.expr()
	SortByExpr{}.expr()
	GroupByExpr{}.expr()
	ReduceExpr{}.expr()
	MapValuesExpr{}.expr()
	AnyExpr{}.expr()
	AllExpr{}.expr()
	CountExpr{}.expr()
	AssignExpr{}.expr()
	NullExpr{}.expr()
	RegexExpr{}.expr()
	ProgramExpr{}.expr()
}

func TestChainExprs(t *testing.T) {
	// 0 args → nil.
	if got := ChainExprs(); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}

	// 1 arg → returns that arg unwrapped.
	single := StringExpr{Value: "a"}
	if got := ChainExprs(single); got != single {
		t.Fatalf("expected single expr returned as-is, got %T", got)
	}

	// 2+ args → ChainedExpr.
	a := StringExpr{Value: "a"}
	b := StringExpr{Value: "b"}
	result := ChainExprs(a, b)
	chained, ok := result.(ChainedExpr)
	if !ok {
		t.Fatalf("expected ChainedExpr, got %T", result)
	}
	if len(chained.Exprs) != 2 {
		t.Fatalf("expected 2 exprs, got %d", len(chained.Exprs))
	}
}

func TestBranchExprs(t *testing.T) {
	a := StringExpr{Value: "a"}
	b := StringExpr{Value: "b"}
	result := BranchExprs(a, b)
	branch, ok := result.(BranchExpr)
	if !ok {
		t.Fatalf("expected BranchExpr, got %T", result)
	}
	if len(branch.Exprs) != 2 {
		t.Fatalf("expected 2 exprs, got %d", len(branch.Exprs))
	}
}

func TestIsType(t *testing.T) {
	s := StringExpr{Value: "x"}
	if !IsType[StringExpr](s) {
		t.Fatal("expected IsType[StringExpr] to be true")
	}
	if IsType[NumberIntExpr](s) {
		t.Fatal("expected IsType[NumberIntExpr] to be false")
	}
}

func TestAsType(t *testing.T) {
	s := StringExpr{Value: "x"}
	got, ok := AsType[StringExpr](s)
	if !ok {
		t.Fatal("expected AsType to succeed")
	}
	if got.Value != "x" {
		t.Fatalf("expected x, got %s", got.Value)
	}

	_, ok = AsType[NumberIntExpr](s)
	if ok {
		t.Fatal("expected AsType to fail for wrong type")
	}
}

func TestLast(t *testing.T) {
	a := StringExpr{Value: "a"}
	b := StringExpr{Value: "b"}
	c := StringExpr{Value: "c"}

	// Chained → last element.
	chain := ChainedExpr{Exprs: Expressions{a, b, c}}
	if got := Last(chain); got != c {
		t.Fatalf("expected last to be c, got %v", got)
	}

	// Non-chained → passthrough.
	if got := Last(a); got != a {
		t.Fatalf("expected passthrough, got %v", got)
	}
}

func TestLastAsType(t *testing.T) {
	a := NumberIntExpr{Value: 1}
	b := StringExpr{Value: "end"}
	chain := ChainedExpr{Exprs: Expressions{a, b}}

	got, ok := LastAsType[StringExpr](chain)
	if !ok {
		t.Fatal("expected LastAsType to succeed")
	}
	if got.Value != "end" {
		t.Fatalf("expected end, got %s", got.Value)
	}

	_, ok = LastAsType[NumberIntExpr](chain)
	if ok {
		t.Fatal("expected LastAsType to fail for wrong type")
	}
}

func TestRemoveLast(t *testing.T) {
	a := StringExpr{Value: "a"}
	b := StringExpr{Value: "b"}
	c := StringExpr{Value: "c"}

	// Chain of 3 → chain of 2.
	chain := ChainedExpr{Exprs: Expressions{a, b, c}}
	result := RemoveLast(chain)
	chained, ok := result.(ChainedExpr)
	if !ok {
		t.Fatalf("expected ChainedExpr, got %T", result)
	}
	if len(chained.Exprs) != 2 {
		t.Fatalf("expected 2 exprs, got %d", len(chained.Exprs))
	}

	// Single expr → nil (RemoveLast on non-chained yields ChainExprs() with 0 args).
	if got := RemoveLast(a); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
