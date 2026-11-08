package execution

import (
	"github.com/tomwright/dasel/v3/model"
	"github.com/tomwright/dasel/v3/selector/ast"
)

// ExecuteOptionFn is a function that can be used to set options on the execution of the selector.
type ExecuteOptionFn func(*Options)

// varScope holds the parameter and ordinary variable bindings for a single
// user-defined function call.
//
// Scopes form a chain: reads fall through to parent scopes (and finally to
// the global Options.Vars), while writes always target the current scope.
// Values are never copied, so mutations made through a referenced object
// (for example via a parameter or $root) still write back to the original
// object.
type varScope struct {
	vars   map[string]*model.Value
	parent *varScope
}

// Options contains the options for the execution of the selector.
type Options struct {
	Funcs    FuncCollection
	Vars     map[string]*model.Value
	Unstable bool

	// userFuncs holds the functions declared with def for the query currently
	// being executed. It is swapped in while a ProgramExpr runs and restored
	// afterwards, so reusing Options across queries never leaks definitions.
	userFuncs map[string]*ast.FunctionExpr
	// scope is the current function call scope chain. It is nil at the top
	// level, where Vars keeps its existing global assignment semantics.
	scope *varScope
	// callChain tracks the user-defined functions currently being executed
	// and is used to reject direct and indirect recursion.
	callChain []string
}

// NewOptions creates a new Options struct with the given options.
func NewOptions(opts ...ExecuteOptionFn) *Options {
	o := &Options{
		Funcs: DefaultFuncCollection,
		Vars:  map[string]*model.Value{},
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(o)
	}
	return o
}

// WithFuncs sets the functions that can be used in the selector.
func WithFuncs(fc FuncCollection) ExecuteOptionFn {
	return func(o *Options) {
		o.Funcs = fc
	}
}

// WithVariable sets a variable for use in the selector.
func WithVariable(key string, val *model.Value) ExecuteOptionFn {
	return func(o *Options) {
		o.Vars[key] = val
	}
}

// WithUnstable allows access to potentially unstable features.
func WithUnstable() ExecuteOptionFn {
	return func(o *Options) {
		o.Unstable = true
	}
}

// WithoutUnstable disallows access to potentially unstable features.
func WithoutUnstable() ExecuteOptionFn {
	return func(o *Options) {
		o.Unstable = false
	}
}

// pushScope enters a function call with the given parameter bindings.
// Every push must be paired with a pop, including when execution errors.
func (o *Options) pushScope(bindings map[string]*model.Value) {
	o.scope = &varScope{
		vars:   bindings,
		parent: o.scope,
	}
}

// popScope leaves the current function call, restoring the caller bindings.
func (o *Options) popScope() {
	if o.scope != nil {
		o.scope = o.scope.parent
	}
}

// lookupVar resolves a variable by walking the scope chain and finally the
// global variables.
func (o *Options) lookupVar(name string) (*model.Value, bool) {
	for scope := o.scope; scope != nil; scope = scope.parent {
		if v, ok := scope.vars[name]; ok {
			return v, true
		}
	}
	v, ok := o.Vars[name]
	return v, ok
}

// setVar writes a variable binding. Inside a function call the binding is
// created or replaced in the current call's scope; at the top level the
// existing global Options.Vars semantics are preserved.
func (o *Options) setVar(name string, val *model.Value) {
	if o.scope != nil {
		o.scope.vars[name] = val
		return
	}
	o.Vars[name] = val
}

// deleteVar removes a binding from the current scope, or from the global
// variables when no function call is active.
func (o *Options) deleteVar(name string) {
	if o.scope != nil {
		delete(o.scope.vars, name)
		return
	}
	delete(o.Vars, name)
}

// withKeyVar sets the $key variable to the given value and returns a function
// that restores the previous value (or removes it if it didn't exist).
func withKeyVar(options *Options, key *model.Value) func() {
	if options.scope != nil {
		prev, had := options.scope.vars["key"]
		options.scope.vars["key"] = key
		return func() {
			if had {
				options.scope.vars["key"] = prev
			} else {
				delete(options.scope.vars, "key")
			}
		}
	}

	prev, had := options.Vars["key"]
	options.Vars["key"] = key
	return func() {
		if had {
			options.Vars["key"] = prev
		} else {
			delete(options.Vars, "key")
		}
	}
}
