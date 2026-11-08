package execution

import (
	"context"
	"fmt"
	"slices"

	"github.com/tomwright/dasel/v3/model"
	"github.com/tomwright/dasel/v3/selector/ast"
)

// programExprExecutor installs the query's user-defined functions for the
// duration of the execution and then runs the main query expression.
//
// The functions are kept on Options rather than registered on the native
// function collection, and the previous state is restored on exit so that
// reusing Options across queries never leaks definitions or local state.
func programExprExecutor(e ast.ProgramExpr) (expressionExecutor, error) {
	return func(ctx context.Context, options *Options, data *model.Value) (*model.Value, error) {
		prevUserFuncs := options.userFuncs
		options.userFuncs = make(map[string]*ast.FunctionExpr, len(e.Functions))
		defer func() {
			options.userFuncs = prevUserFuncs
		}()

		for i := range e.Functions {
			fn := e.Functions[i]
			if _, ok := options.Funcs.Get(fn.Name); ok {
				return nil, fmt.Errorf("function %q cannot be defined: a native function with the same name already exists", fn.Name)
			}
			options.userFuncs[fn.Name] = &fn
		}

		return ExecuteAST(ctx, e.Expr, data, options)
	}, nil
}

// userFuncCallExecutor executes a user-defined function declared with def.
func userFuncCallExecutor(options *Options, fn *ast.FunctionExpr, argsE ast.Expressions) (expressionExecutor, error) {
	return func(ctx context.Context, options *Options, data *model.Value) (*model.Value, error) {
		ctx = WithExecutorID(ctx, "userFuncCallExpr")

		// Reject recursion before evaluating any argument so a recursive call
		// fails fast with a clear error instead of running until the stack
		// overflows.
		if slices.Contains(options.callChain, fn.Name) {
			return nil, fmt.Errorf("recursive call to function %q is not allowed", fn.Name)
		}

		// Every argument is evaluated exactly once on the caller's current
		// data. The resulting values are shared by reference with the body:
		// they are never copied.
		args, err := prepareArgs(ctx, options, data, argsE)
		if err != nil {
			return nil, fmt.Errorf("error preparing arguments for function %q: %w", fn.Name, err)
		}
		if len(args) != len(fn.Params) {
			return nil, fmt.Errorf("func %q expects exactly %d arguments, got %d", fn.Name, len(fn.Params), len(args))
		}

		options.callChain = append(options.callChain, fn.Name)
		defer func() {
			options.callChain = options.callChain[:len(options.callChain)-1]
		}()

		bindings := make(map[string]*model.Value, len(fn.Params))
		for i, param := range fn.Params {
			bindings[param] = args[i]
		}

		// The call gets its own parameter and variable bindings. Reads fall
		// through to the caller's scope; writes only affect this call. The
		// deferred pop restores the caller bindings even when the body fails.
		options.pushScope(bindings)
		defer options.popScope()

		res, err := ExecuteAST(ctx, fn.Body, data, options)
		if err != nil {
			return nil, fmt.Errorf("error executing function %q: %w", fn.Name, err)
		}
		return res, nil
	}, nil
}
