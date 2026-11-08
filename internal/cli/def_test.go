package cli_test

import (
	"strings"
	"testing"
)

func TestQuery_Def(t *testing.T) {
	t.Run("declared function used in map", runTest(testCase{
		args: []string{
			"-i", "json", "-o", "json",
			`def label($suffix) { name + $suffix }; items.map(label("!"))`,
		},
		in:     []byte(`{"items":[{"name":"a"},{"name":"b"}]}`),
		stdout: []byte("[\n    \"a!\",\n    \"b!\"\n]\n"),
	}))

	t.Run("function uses cli variable", runTest(testCase{
		args: []string{
			"-i", "json", "-o", "json",
			"--var", "base=json:10",
			`def addBase($n) { $n + $base }; addBase(5)`,
		},
		in:     []byte(`null`),
		stdout: []byte("15\n"),
	}))

	t.Run("function chaining and native calls", runTest(testCase{
		args: []string{
			"-i", "json", "-o", "json",
			`def shout($s) { toUpper($s) }; def shoutTwice($s) { shout(shout($s)) }; map(shoutTwice($this))`,
		},
		in:     []byte(`["a","b"]`),
		stdout: []byte("[\n    \"A\",\n    \"B\"\n]\n"),
	}))

	t.Run("recursion returns a clear error", func(t *testing.T) {
		_, _, err := runDasel(
			[]string{"-i", "json", "-o", "json", `def f() { f() }; f()`},
			[]byte(`null`),
		)
		if err == nil {
			t.Fatal("expected recursion error")
		}
		if !strings.Contains(err.Error(), `recursive call to function "f" is not allowed`) {
			t.Fatalf("unexpected error: %s", err)
		}
	})

	t.Run("def field remains accessible as data", runTest(testCase{
		args:   []string{"-i", "json", "-o", "json", `def`},
		in:     []byte(`{"def": 1}`),
		stdout: []byte("1\n"),
	}))
}
