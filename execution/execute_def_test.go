package execution_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tomwright/dasel/v3/execution"
	"github.com/tomwright/dasel/v3/model"
	"github.com/tomwright/dasel/v3/model/orderedmap"
)

func TestDef_HappyPath(t *testing.T) {
	t.Run("basic example with map", testCase{
		inFn: func() *model.Value {
			return model.NewValue(orderedmap.NewMap().Set("items", []any{
				orderedmap.NewMap().Set("name", "a"),
				orderedmap.NewMap().Set("name", "b"),
			}))
		},
		s:   `def label($suffix) { name + $suffix }; items.map(label("!"))`,
		out: model.NewValue([]any{"a!", "b!"}),
	}.run)

	t.Run("declaration does not consume input and body starts at current data", testCase{
		in:  model.NewIntValue(41),
		s:   `def answer() { $this + 1 }; answer()`,
		out: model.NewIntValue(42),
	}.run)

	t.Run("zero parameter function", testCase{
		in:  model.NewNullValue(),
		s:   `def one() { 1 }; one()`,
		out: model.NewIntValue(1),
	}.run)

	t.Run("multiple parameters", testCase{
		in:  model.NewNullValue(),
		s:   `def combine($a, $b) { $a + $b }; combine(20, 22)`,
		out: model.NewIntValue(42),
	}.run)

	t.Run("semicolon chained body returns last expression", testCase{
		in:  model.NewNullValue(),
		s:   `def calc($x) { $y = $x + 1; $y * 2 }; calc(10)`,
		out: model.NewIntValue(22),
	}.run)

	t.Run("body can call native functions", testCase{
		in:  model.NewNullValue(),
		s:   `def up($s) { toUpper($s) }; up("abc")`,
		out: model.NewStringValue("ABC"),
	}.run)

	t.Run("body can call other declared functions", testCase{
		in:  model.NewNullValue(),
		s:   `def inc($x) { $x + 1 }; def add2($x) { inc(inc($x)) }; add2(10)`,
		out: model.NewIntValue(12),
	}.run)

	t.Run("arguments are evaluated on the caller current data", testCase{
		in:  model.NewValue([]any{
			orderedmap.NewMap().Set("name", "a"),
			orderedmap.NewMap().Set("name", "b"),
		}),
		s:   `def wrap($p) { $p + "!" }; map(wrap(name))`,
		out: model.NewValue([]any{"a!", "b!"}),
	}.run)

	t.Run("each argument is evaluated exactly once", func(t *testing.T) {
		// side() mutates a root field, so any double evaluation of the
		// argument would be observable in the resulting counter.
		in := model.NewValue(orderedmap.NewMap().Set("hits", int64(0)))
		opts := execution.NewOptions(execution.WithVariable("root", in))
		res, err := execution.ExecuteSelector(
			context.Background(),
			`def side() { $root.hits = $root.hits + 1; $root.hits }; def once($arg) { $arg }; once(side()); $root.hits`,
			in,
			opts,
		)
		if err != nil {
			t.Fatal(err)
		}
		equal, err := res.EqualTypeValue(model.NewIntValue(1))
		if err != nil {
			t.Fatal(err)
		}
		if !equal {
			t.Errorf("expected argument to be evaluated once, got %s", res.String())
		}
	})

	t.Run("spread arguments are bound positionally", testCase{
		in: model.NewNullValue(),
		s:  `def combine($a, $b) { $a + $b }; combine($vals...)`,
		out: model.NewIntValue(3),
		opts: []execution.ExecuteOptionFn{
			execution.WithVariable("vals", model.NewValue([]any{1, 2})),
		},
	}.run)

	t.Run("function can read caller variables", testCase{
		in: model.NewNullValue(),
		s:  `def f() { $base + 5 }; f()`,
		out: model.NewIntValue(15),
		opts: []execution.ExecuteOptionFn{
			execution.WithVariable("base", model.NewIntValue(10)),
		},
	}.run)

	t.Run("nested call shadows same named variable without changing outer binding", testCase{
		in:  model.NewNullValue(),
		s:   `def outer() { $v = 1; inner(); $v }; def inner() { $v = 2; $v }; outer()`,
		out: model.NewIntValue(1),
	}.run)

	t.Run("assignment inside a function does not leak to globals", func(t *testing.T) {
		in := model.NewNullValue()
		res, err := execution.ExecuteSelector(
			context.Background(),
			`def f() { $g = 99 }; $g = 10; f(); $g`,
			in,
			execution.NewOptions(),
		)
		if err != nil {
			t.Fatal(err)
		}
		equal, err := res.EqualTypeValue(model.NewIntValue(10))
		if err != nil {
			t.Fatal(err)
		}
		if !equal {
			t.Errorf("expected global $g to remain 10, got %s", res.String())
		}
	})

	t.Run("reassigning a parameter does not change the caller binding", func(t *testing.T) {
		in := model.NewNullValue()
		res, err := execution.ExecuteSelector(
			context.Background(),
			`def f($x) { $x = 99; $x }; $a = 1; f($a); $a`,
			in,
			execution.NewOptions(),
		)
		if err != nil {
			t.Fatal(err)
		}
		equal, err := res.EqualTypeValue(model.NewIntValue(1))
		if err != nil {
			t.Fatal(err)
		}
		if !equal {
			t.Errorf("expected caller $a to remain 1, got %s", res.String())
		}
	})

	t.Run("field mutation through parameter writes back to the original object", func(t *testing.T) {
		in := model.NewValue(orderedmap.NewMap().Set("name", "a"))
		opts := execution.NewOptions(execution.WithVariable("root", in))
		res, err := execution.ExecuteSelector(
			context.Background(),
			`def rename($o) { $o.name = "z" }; rename($root); $root.name`,
			in,
			opts,
		)
		if err != nil {
			t.Fatal(err)
		}
		if equal, err := res.EqualTypeValue(model.NewStringValue("z")); err != nil || !equal {
			t.Fatalf("unexpected result: %s (err: %v)", res.String(), err)
		}
		if equal, err := in.EqualTypeValue(model.NewValue(orderedmap.NewMap().Set("name", "z"))); err != nil || !equal {
			t.Fatalf("expected original object to be mutated, got %s", in.String())
		}
	})

	t.Run("field mutation through $root writes back to the original object", func(t *testing.T) {
		in := model.NewValue(orderedmap.NewMap().Set("hits", int64(5)))
		opts := execution.NewOptions(execution.WithVariable("root", in))
		res, err := execution.ExecuteSelector(
			context.Background(),
			`def bump() { $root.hits = $root.hits + 1 }; bump(); $root.hits`,
			in,
			opts,
		)
		if err != nil {
			t.Fatal(err)
		}
		if equal, err := res.EqualTypeValue(model.NewIntValue(6)); err != nil || !equal {
			t.Fatalf("unexpected result: %s (err: %v)", res.String(), err)
		}
	})

	t.Run("$this reflects the current node inside the body", testCase{
		in:  model.NewValue([]any{10, 20, 30}),
		s:   `def self() { $this }; map(self())`,
		out: model.NewValue([]any{10, 20, 30}),
	}.run)

	t.Run("$key keeps reflecting the iteration context inside the body", testCase{
		in:  model.NewValue([]any{10, 20, 30}),
		s:   `def idx() { $key }; map(idx())`,
		out: model.NewValue([]any{0, 1, 2}),
	}.run)

	t.Run("def field name remains a plain data property", testCase{
		in:  model.NewValue(orderedmap.NewMap().Set("def", int64(1))),
		s:   `def`,
		out: model.NewIntValue(1),
	}.run)
}

func TestDef_Errors(t *testing.T) {
	cases := []struct {
		name    string
		s       string
		wantErr string
	}{
		{
			name:    "duplicate definition",
			s:       `def f() { 1 }; def f() { 2 }; f()`,
			wantErr: "defined more than once",
		},
		{
			name:    "duplicate parameter",
			s:       `def f($x, $x) { $x }; f(1)`,
			wantErr: "duplicate parameter",
		},
		{
			name:    "name clashes with native function",
			s:       `def len() { 1 }; len()`,
			wantErr: "native function with the same name already exists",
		},
		{
			name:    "too few arguments",
			s:       `def f($a) { $a }; f()`,
			wantErr: `expects exactly 1 arguments, got 0`,
		},
		{
			name:    "too many arguments",
			s:       `def f($a) { $a }; f(1, 2)`,
			wantErr: `expects exactly 1 arguments, got 2`,
		},
		{
			name:    "direct recursion",
			s:       `def f() { f() }; f()`,
			wantErr: `recursive call to function "f" is not allowed`,
		},
		{
			name:    "indirect recursion",
			s:       `def a() { b() }; def b() { a() }; a()`,
			wantErr: `recursive call to function "a" is not allowed`,
		},
		{
			name:    "unknown function stays unknown",
			s:       `def f() { 1 }; g()`,
			wantErr: `unknown function: "g"`,
		},
		{
			name:    "body error message is preserved",
			s:       `def f() { $undefined_var_xyz }; f()`,
			wantErr: "variable undefined_var_xyz not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execution.ExecuteSelector(
				context.Background(),
				tc.s,
				model.NewNullValue(),
				execution.NewOptions(),
			)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %s", tc.wantErr, err)
			}
		})
	}
}

func TestDef_OptionsReuseAndIsolation(t *testing.T) {
	ctx := context.Background()
	in := model.NewNullValue()
	opts := execution.NewOptions()

	// A failing query (recursion) must not leave any state behind.
	if _, err := execution.ExecuteSelector(ctx, `def f() { f() }; f()`, in, opts); err == nil {
		t.Fatal("expected recursion error")
	} else if !strings.Contains(err.Error(), "recursive call") {
		t.Fatalf("unexpected error: %s", err)
	}

	// The corrected query runs fine with the same Options.
	res, err := execution.ExecuteSelector(ctx, `def f() { 42 }; f()`, in, opts)
	if err != nil {
		t.Fatal(err)
	}
	if equal, err := res.EqualTypeValue(model.NewIntValue(42)); err != nil || !equal {
		t.Fatalf("unexpected corrected query result: %s (err: %v)", res.String(), err)
	}

	// Definitions must not leak into subsequent queries.
	if _, err := execution.ExecuteSelector(ctx, `f()`, in, opts); err == nil {
		t.Fatal("expected leaked function to be unknown")
	} else if !strings.Contains(err.Error(), `unknown function: "f"`) {
		t.Fatalf("unexpected error: %s", err)
	}

	// Function local variables must not leak into Options.Vars.
	if _, err := execution.ExecuteSelector(ctx, `def g() { $tmp = 1; $tmp }; g()`, in, opts); err != nil {
		t.Fatal(err)
	}
	if _, ok := opts.Vars["tmp"]; ok {
		t.Fatal("function local variable $tmp leaked into Options.Vars")
	}

	// A failed body must restore the caller bindings while keeping the message.
	if _, err := execution.ExecuteSelector(
		ctx,
		`def f() { $g = 99; $undefined_xyz }; $g = 1; f()`,
		in,
		opts,
	); err == nil {
		t.Fatal("expected body error")
	} else if !strings.Contains(err.Error(), "variable undefined_xyz not found") {
		t.Fatalf("unexpected error: %s", err)
	}
	res, err = execution.ExecuteSelector(ctx, `$g`, in, opts)
	if err != nil {
		t.Fatal(err)
	}
	if equal, err := res.EqualTypeValue(model.NewIntValue(1)); err != nil || !equal {
		t.Fatalf("expected caller binding $g=1 to be restored, got %s", res.String())
	}

	// Top level global assignment semantics remain unchanged.
	if _, err := execution.ExecuteSelector(ctx, `$global = 5`, in, opts); err != nil {
		t.Fatal(err)
	}
	v, ok := opts.Vars["global"]
	if !ok {
		t.Fatal("expected global variable to persist in Options.Vars")
	}
	if equal, err := v.EqualTypeValue(model.NewIntValue(5)); err != nil || !equal {
		t.Fatalf("unexpected global value: %s", v.String())
	}
}
