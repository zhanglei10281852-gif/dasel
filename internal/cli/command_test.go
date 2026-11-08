package cli_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tomwright/dasel/v3/internal/cli"
)

func runDasel(args []string, in []byte) ([]byte, []byte, error) {
	stdOut := bytes.NewBuffer([]byte{})
	stdErr := bytes.NewBuffer([]byte{})
	stdIn := bytes.NewReader(in)

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	os.Args = append([]string{"dasel", "query"}, args...)

	_, err := cli.Run(stdIn, stdOut, stdErr)

	return stdOut.Bytes(), stdErr.Bytes(), err
}

type testCase struct {
	args   []string
	in     []byte
	stdout []byte
	stderr []byte
	err    error
}

func runTest(tc testCase) func(t *testing.T) {
	return func(t *testing.T) {
		if tc.stdout == nil {
			tc.stdout = []byte{}
		}
		if tc.stderr == nil {
			tc.stderr = []byte{}
		}

		gotStdOut, gotStdErr, gotErr := runDasel(tc.args, tc.in)
		if !errors.Is(gotErr, tc.err) && !errors.Is(tc.err, gotErr) {
			t.Errorf("expected error %v, got %v", tc.err, gotErr)
			return
		}

		if !reflect.DeepEqual(tc.stderr, gotStdErr) {
			t.Errorf("expected stderr %s, got %s", string(tc.stderr), string(gotStdErr))
		}

		if !reflect.DeepEqual(tc.stdout, gotStdOut) {
			t.Errorf("expected stdout %s, got %s", string(tc.stdout), string(gotStdOut))
		}
	}
}

func TestRun(t *testing.T) {
	t.Run("complex set", func(t *testing.T) {
		t.Run("set nested with spread", runTest(testCase{
			args: []string{"-i", "json", "-o", "json", "--root", `user = {user..., name: {"first": $this.user.name, "last": "Doe"}}`},
			in:   []byte(`{"user": {"name": "John"}}`),
			stdout: []byte(`{
    "user": {
        "name": {
            "first": "John",
            "last": "Doe"
        }
    }
}
`),
			stderr: nil,
			err:    nil,
		}))
		t.Run("set nested", runTest(testCase{
			args: []string{"-i", "json", "-o", "json", "--root", `user.name = {"first": user.name, "last": "Doe"}`},
			in:   []byte(`{"user": {"name": "John"}}`),
			stdout: []byte(`{
    "user": {
        "name": {
            "first": "John",
            "last": "Doe"
        }
    }
}
`),
			stderr: nil,
			err:    nil,
		}))
		t.Run("set nested with localised group", runTest(testCase{
			args: []string{"-i", "json", "-o", "json", "--root", `user.(name = {"first": name, "last": "Doe"})`},
			in:   []byte(`{"user": {"name": "John"}}`),
			stdout: []byte(`{
    "user": {
        "name": {
            "first": "John",
            "last": "Doe"
        }
    }
}
`),
			stderr: nil,
			err:    nil,
		}))
		t.Run("set recursive descent", func(t *testing.T) {

			t.Run("property", runTest(testCase{
				args: []string{"-i", "json", "-o", "json", "--root", `$root..x.each($this = $this+1)`},
				in:   []byte(`[{"x":1},{"x":2},{"x":3}]`),
				stdout: []byte(`[
    {
        "x": 2
    },
    {
        "x": 3
    },
    {
        "x": 4
    }
]
`),
				stderr: nil,
				err:    nil,
			}))

			t.Run("index", runTest(testCase{
				args: []string{"-i", "json", "-o", "json", "--root", `$root..[1].each($this = $this+1)`},
				in:   []byte(`[ {"x":[1,2,3]} , {"y":[4,5,6]} , {"z":[7,8,9]} ]`),
				stdout: []byte(`[
    {
        "x": [
            1,
            3,
            3
        ]
    },
    {
        "y": [
            4,
            6,
            6
        ]
    },
    {
        "z": [
            7,
            9,
            9
        ]
    }
]
`),
				stderr: nil,
				err:    nil,
			}))

			t.Run("wildcard", runTest(testCase{
				args: []string{"-i", "json", "-o", "json", "--root", `$root..*.each($this = 4)`},
				in:   []byte(`[{"x":1},{"x":2},{"x":3}]`),
				stdout: []byte(`[
    {
        "x": 4
    },
    {
        "x": 4
    },
    {
        "x": 4
    }
]
`),
				stderr: nil,
				err:    nil,
			}))

		})
		t.Run("create object with empty stdin", runTest(testCase{
			args: []string{`{"name":"Tom"}`},
			in:   []byte{},
			stdout: []byte(`{
    "name": "Tom"
}
`),
			stderr: nil,
			err:    nil,
		}))
	})
	t.Run("set search", runTest(testCase{
		args: []string{"-i", "json", "-o", "json", "--root", `search(has("x")).each(x = x+1)`},
		in:   []byte(`[{"x":1},{"x":2},{"x":3}]`),
		stdout: []byte(`[
    {
        "x": 2
    },
    {
        "x": 3
    },
    {
        "x": 4
    }
]
`),
		stderr: nil,
		err:    nil,
	}))
	t.Run("recursive descent", func(t *testing.T) {
		t.Run("wildcard", runTest(testCase{
			args: []string{"-i", "json", `..*`},
			in: []byte(`{
  "user": {
    "name": "Alice",
    "roles": ["admin", "editor"],
    "meta": {
      "active": true,
      "score": 42
    }
  },
  "tags": ["x", "y"],
  "count": 10
}`),
			stdout: []byte(`[
    "Alice",
    "admin",
    "editor",
    true,
    42,
    "x",
    "y",
    10
]
`),
			stderr: nil,
			err:    nil,
		}))

		t.Run("property", runTest(testCase{
			args: []string{"-i", "json", `..name`},
			in: []byte(`{
  "user": {
    "name": "Alice",
    "roles": ["admin", "editor"],
    "meta": {
      "active": true,
      "score": 42
    }
  },
  "tags": ["x", "y"],
  "count": 10
}`),
			stdout: []byte(`[
    "Alice"
]
`),
			stderr: nil,
			err:    nil,
		}))

		t.Run("property2", runTest(testCase{
			args: []string{"-i", "json", `..name`},
			in:   []byte(`[{"name":"Tom"}, {"name":"Jim"}, {"foo": "Bar"}]`),
			stdout: []byte(`[
    "Tom",
    "Jim"
]
`),
			stderr: nil,
			err:    nil,
		}))

		t.Run("index", runTest(testCase{
			args: []string{"-i", "json", `..[0]`},
			in: []byte(`{
  "user": {
    "name": "Alice",
    "roles": ["admin", "editor"],
    "meta": {
      "active": true,
      "score": 42
    }
  },
  "tags": ["x", "y"],
  "count": 10
}`),
			stdout: []byte(`[
    "admin",
    "x"
]
`),
			stderr: nil,
			err:    nil,
		}))
	})
	t.Run("any", func(t *testing.T) {
		t.Run("true result", runTest(testCase{
			args:   []string{"-i", "json", `users.any(age > 25)`},
			in:     []byte(`{"users": [{"name": "Alice", "age": 20}, {"name": "Bob", "age": 30}]}`),
			stdout: []byte("true\n"),
			stderr: nil,
			err:    nil,
		}))
		t.Run("false result", runTest(testCase{
			args:   []string{"-i", "json", `users.any(age > 50)`},
			in:     []byte(`{"users": [{"name": "Alice", "age": 20}, {"name": "Bob", "age": 30}]}`),
			stdout: []byte("false\n"),
			stderr: nil,
			err:    nil,
		}))
	})
	t.Run("all", func(t *testing.T) {
		t.Run("true result", runTest(testCase{
			args:   []string{"-i", "json", `users.all(age > 18)`},
			in:     []byte(`{"users": [{"name": "Alice", "age": 20}, {"name": "Bob", "age": 30}]}`),
			stdout: []byte("true\n"),
			stderr: nil,
			err:    nil,
		}))
		t.Run("false result", runTest(testCase{
			args:   []string{"-i", "json", `users.all(age > 25)`},
			in:     []byte(`{"users": [{"name": "Alice", "age": 20}, {"name": "Bob", "age": 30}]}`),
			stdout: []byte("false\n"),
			stderr: nil,
			err:    nil,
		}))
	})
	t.Run("ternary", func(t *testing.T) {
		t.Run("true literal", runTest(testCase{
			args:   []string{`true ? "yes" : "no"`},
			in:     []byte{},
			stdout: []byte("\"yes\"\n"),
			err:    nil,
		}))
		t.Run("false literal", runTest(testCase{
			args:   []string{`false ? "yes" : "no"`},
			in:     []byte{},
			stdout: []byte("\"no\"\n"),
			err:    nil,
		}))
		t.Run("with json data true", runTest(testCase{
			args:   []string{"-i", "json", `age >= 18 ? "adult" : "minor"`},
			in:     []byte(`{"age": 25}`),
			stdout: []byte("\"adult\"\n"),
			err:    nil,
		}))
		t.Run("with json data false", runTest(testCase{
			args:   []string{"-i", "json", `age >= 18 ? "adult" : "minor"`},
			in:     []byte(`{"age": 10}`),
			stdout: []byte("\"minor\"\n"),
			err:    nil,
		}))
		t.Run("nested ternary", runTest(testCase{
			args:   []string{`true ? (false ? "a" : "b") : "c"`},
			in:     []byte{},
			stdout: []byte("\"b\"\n"),
			err:    nil,
		}))
		t.Run("with comparison", runTest(testCase{
			args:   []string{"-i", "json", `score >= 90 ? "A" : (score >= 80 ? "B" : "C")`},
			in:     []byte(`{"score": 85}`),
			stdout: []byte("\"B\"\n"),
			err:    nil,
		}))
		t.Run("ternary in map", runTest(testCase{
			args: []string{"-i", "json", `items.map(val >= 5 ? "big" : "small")`},
			in:   []byte(`{"items": [{"val": 10}, {"val": 3}, {"val": 7}]}`),
			stdout: []byte(`[
    "big",
    "small",
    "big"
]
`),
			err: nil,
		}))
		t.Run("ternary returns number", runTest(testCase{
			args:   []string{`true ? 42 : 0`},
			in:     []byte{},
			stdout: []byte("42\n"),
			err:    nil,
		}))
		t.Run("ternary with arithmetic", runTest(testCase{
			args:   []string{`true ? 1 + 2 : 3 + 4`},
			in:     []byte{},
			stdout: []byte("3\n"),
			err:    nil,
		}))
		t.Run("ternary with logical operators", runTest(testCase{
			args:   []string{"-i", "json", `a > 1 && b > 5 ? "both" : "nope"`},
			in:     []byte(`{"a": 5, "b": 10}`),
			stdout: []byte("\"both\"\n"),
			err:    nil,
		}))
		t.Run("chained property in condition", runTest(testCase{
			args:   []string{"-i", "json", `user.active ? "yes" : "no"`},
			in:     []byte(`{"user": {"active": true}}`),
			stdout: []byte("\"yes\"\n"),
			err:    nil,
		}))
		t.Run("ternary with json output", runTest(testCase{
			args: []string{"-i", "json", "-o", "json", `active ? {status: "on"} : {status: "off"}`},
			in:   []byte(`{"active": true}`),
			stdout: []byte(`{
    "status": "on"
}
`),
			err: nil,
		}))
		t.Run("ternary returns true", runTest(testCase{
			args:   []string{`true ? true : false`},
			in:     []byte{},
			stdout: []byte("true\n"),
			err:    nil,
		}))
	})
	t.Run("compact", func(t *testing.T) {
		t.Run("json", runTest(testCase{
			args:   []string{"-i", "json", "-o", "json", "--compact"},
			in:     []byte(`{"name": "Tom", "age": 30}`),
			stdout: []byte("{\"name\":\"Tom\",\"age\":30}\n"),
			err:    nil,
		}))
		t.Run("json nested", runTest(testCase{
			args:   []string{"-i", "json", "-o", "json", "--compact"},
			in:     []byte(`{"user": {"name": "Tom"}, "items": [1, 2, 3]}`),
			stdout: []byte("{\"user\":{\"name\":\"Tom\"},\"items\":[1,2,3]}\n"),
			err:    nil,
		}))
	})
	t.Run("merge", func(t *testing.T) {
		t.Run("deep", runTest(testCase{
			args: []string{"-i", "json", "-o", "json",
				`merge({"a": {"x": 1, "y": 2}}, {"a": {"y": 3, "z": 4}})`},
			in: []byte{},
			stdout: []byte("{\n    \"a\": {\n        \"x\": 1,\n        \"y\": 3,\n        \"z\": 4\n    }\n}\n"),
		}))
	})
	t.Run("count", func(t *testing.T) {
		t.Run("some match", runTest(testCase{
			args:   []string{"-i", "json", `users.count(age > 25)`},
			in:     []byte(`{"users": [{"name": "Alice", "age": 20}, {"name": "Bob", "age": 30}, {"name": "Charlie", "age": 35}]}`),
			stdout: []byte("2\n"),
			stderr: nil,
			err:    nil,
		}))
		t.Run("none match", runTest(testCase{
			args:   []string{"-i", "json", `users.count(age > 50)`},
			in:     []byte(`{"users": [{"name": "Alice", "age": 20}, {"name": "Bob", "age": 30}]}`),
			stdout: []byte("0\n"),
			stderr: nil,
			err:    nil,
		}))
	})
}

// runDaselIO runs dasel with custom reader/writer, for stream tests that
// need to observe read/write failures.
func runDaselIO(args []string, in io.Reader, out io.Writer) error {
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()
	os.Args = append([]string{"dasel", "query"}, args...)
	_, err := cli.Run(in, out, bytes.NewBuffer(nil))
	return err
}

type trackingReader struct {
	data []byte
	read bool
}

func (r *trackingReader) Read(p []byte) (int, error) {
	r.read = true
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

var errStreamTestWrite = errors.New("stream test write failure")

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errStreamTestWrite }

// acceptOneWriter accepts the first write then fails all subsequent writes.
type acceptOneWriter struct {
	writes int
}

func (w *acceptOneWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		return len(p), nil
	}
	return 0, errStreamTestWrite
}

func TestStream(t *testing.T) {
	t.Run("queries each record in order", runTest(testCase{
		args:   []string{"--stream", `name`},
		in:     []byte("{\"name\":\"a\"}\n{\"name\":\"b\"}\n{\"name\":\"c\"}\n"),
		stdout: []byte("\"a\"\n\"b\"\n\"c\"\n"),
	}))

	t.Run("defaults to json without explicit formats", runTest(testCase{
		args:   []string{"--stream", `name`},
		in:     []byte("{\"name\":\"a\"}{\"name\":\"b\"}"),
		stdout: []byte("\"a\"\n\"b\"\n"),
	}))

	t.Run("empty stream executes no query and outputs nothing", runTest(testCase{
		args:   []string{"--stream", `name`},
		in:     []byte{},
		stdout: []byte{},
	}))

	t.Run("whitespace only stream outputs nothing", runTest(testCase{
		args:   []string{"--stream", `name`},
		in:     []byte("  \n\t\r\n"),
		stdout: []byte{},
	}))

	t.Run("records can span multiple lines", runTest(testCase{
		args:   []string{"--stream", `name`},
		in:     []byte("{\n  \"name\": \"a\"\n}\n{\n  \"name\": \"b\"\n}\n"),
		stdout: []byte("\"a\"\n\"b\"\n"),
	}))

	t.Run("empty query passes records through", runTest(testCase{
		args: []string{"--stream"},
		in:   []byte("[1, 2][3, 4]"),
		stdout: []byte(`[
    1,
    2
]
[
    3,
    4
]
`),
	}))

	t.Run("array is one record even with elements on lines", runTest(testCase{
		args:   []string{"--stream", `$root[0]`},
		in:     []byte("[\n  1,\n  2\n]\n[\n  3,\n  4\n]\n"),
		stdout: []byte("1\n3\n"),
	}))

	t.Run("scalar records", runTest(testCase{
		args:   []string{"--stream"},
		in:     []byte("1 2 true false \"x\" null\n"),
		stdout: []byte("1\n2\ntrue\nfalse\n\"x\"\nnull\n"),
	}))

	t.Run("compact output", runTest(testCase{
		args:   []string{"--stream", "--compact"},
		in:     []byte("{\"name\":\"a\"}\n{\"name\":\"b\"}\n"),
		stdout: []byte("{\"name\":\"a\"}\n{\"name\":\"b\"}\n"),
	}))

	t.Run("root and field modifications stay within record", runTest(testCase{
		args:   []string{"--stream", "--root", `a = 2`},
		in:     []byte("{\"a\":1}\n{\"a\":1}\n"),
		stdout: []byte("{\n    \"a\": 2\n}\n{\n    \"a\": 2\n}\n"),
	}))

	t.Run("var reassignment resets for every record", runTest(testCase{
		args:   []string{"--stream", "--var", "c=json:0", `$c = $c + 1`},
		in:     []byte("{}\n{}\n{}\n"),
		stdout: []byte("1\n1\n1\n"),
	}))

	t.Run("nested var mutations reset for every record", runTest(testCase{
		args:   []string{"--stream", "--var", "x=json:{\"n\":1}", `$x.n = $x.n + 1`},
		in:     []byte("{}\n{}\n{}\n"),
		stdout: []byte("2\n2\n2\n"),
	}))

	t.Run("new vars do not leak into later records", runTest(testCase{
		// The first statement creates $z; a fresh vars map per record means
		// the second statement still resolves against the just-created var.
		args:   []string{"--stream", `$z = "v"; $z`},
		in:     []byte("{}\n{}\n"),
		stdout: []byte("\"v\"\n\"v\"\n"),
	}))

	t.Run("spread keeps multi-document output per record", runTest(testCase{
		args:   []string{"--stream", `...`},
		in:     []byte("[1, 2]\n[3, 4]\n"),
		stdout: []byte("1\n\n2\n3\n\n4\n"),
	}))

	t.Run("branch result writes multiple documents", runTest(testCase{
		args:   []string{"--stream", "--unstable", `branch(...)`},
		in:     []byte("[1, 2]\n[3]\n"),
		stdout: []byte("1\n\n2\n3\n"),
	}))

	t.Run("parse failure keeps prior results and reports record index", func(t *testing.T) {
		out, _, err := runDasel([]string{"--stream", `a`},
			[]byte("{\"a\":1}\n{\"a\":2}\n{oops\n{\"a\":4}\n"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "record 3") {
			t.Fatalf("expected error to reference record 3, got: %v", err)
		}
		if string(out) != "1\n2\n" {
			t.Fatalf("expected only prior record results, got: %q", string(out))
		}
	})

	t.Run("query failure keeps prior results and reports record index", func(t *testing.T) {
		out, _, err := runDasel([]string{"--stream", `a`},
			[]byte("{\"a\":1}\n{\"a\":2}\n5\n{\"a\":4}\n"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "record 3") {
			t.Fatalf("expected error to reference record 3, got: %v", err)
		}
		if string(out) != "1\n2\n" {
			t.Fatalf("expected only prior record results, got: %q", string(out))
		}
	})

	t.Run("failed record emits no partial output", func(t *testing.T) {
		out, _, err := runDasel([]string{"--stream", `a`},
			[]byte("{\"a\":1}\n{oops\n"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if string(out) != "1\n" {
			t.Fatalf("expected no output for the failed record, got: %q", string(out))
		}
	})

	t.Run("output write failure stops further records", func(t *testing.T) {
		w := &acceptOneWriter{}
		err := runDaselIO([]string{"--stream", `a`},
			strings.NewReader("{\"a\":1}\n{\"a\":2}\n{\"a\":3}\n"), w)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errStreamTestWrite) {
			t.Fatalf("expected write failure, got: %v", err)
		}
		if !strings.Contains(err.Error(), "record 2") {
			t.Fatalf("expected error to reference record 2, got: %v", err)
		}
		if w.writes != 2 {
			t.Fatalf("expected processing to stop after record 2, got %d write attempts", w.writes)
		}
	})

	t.Run("immediate output write failure reports record 1", func(t *testing.T) {
		err := runDaselIO([]string{"--stream", `a`},
			strings.NewReader("{\"a\":1}\n{\"a\":2}\n"), errorWriter{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errStreamTestWrite) {
			t.Fatalf("expected write failure, got: %v", err)
		}
		if !strings.Contains(err.Error(), "record 1") {
			t.Fatalf("expected error to reference record 1, got: %v", err)
		}
	})

	t.Run("unsupported input format errors before reading input", func(t *testing.T) {
		r := &trackingReader{data: []byte("{\"a\":1}\n")}
		err := runDaselIO([]string{"--stream", "-i", "yaml", `a`}, r, bytes.NewBuffer(nil))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "json") {
			t.Fatalf("expected error to mention json, got: %v", err)
		}
		if r.read {
			t.Fatal("expected stdin not to be read before the format error")
		}
	})

	t.Run("unsupported output format errors before reading input", func(t *testing.T) {
		r := &trackingReader{data: []byte("{\"a\":1}\n")}
		err := runDaselIO([]string{"--stream", "-o", "yaml", `a`}, r, bytes.NewBuffer(nil))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "json") {
			t.Fatalf("expected error to mention json, got: %v", err)
		}
		if r.read {
			t.Fatal("expected stdin not to be read before the format error")
		}
	})

	t.Run("interactive combination errors before reading input", func(t *testing.T) {
		r := &trackingReader{data: []byte("{\"a\":1}\n")}
		err := runDaselIO([]string{"--stream", "--it", `a`}, r, bytes.NewBuffer(nil))
		if !errors.Is(err, cli.ErrStreamInteractive) {
			t.Fatalf("expected ErrStreamInteractive, got: %v", err)
		}
		if r.read {
			t.Fatal("expected stdin not to be read before the interactive error")
		}
	})
}

func TestNonStreamBehaviourUnchanged(t *testing.T) {
	t.Run("empty input outputs null", runTest(testCase{
		args:   []string{"-i", "json", "-o", "json"},
		in:     []byte{},
		stdout: []byte("null\n"),
	}))

	t.Run("ndjson remains a single multi-document batch", runTest(testCase{
		args:   []string{"-i", "json", "-o", "json", `a`},
		in:     []byte("{\"a\":1}\n{\"a\":2}\n"),
		stdout: []byte("1\n\n2\n"),
	}))
}
