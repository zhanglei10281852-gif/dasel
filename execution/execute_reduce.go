package execution

import (
	"context"
	"fmt"

	"github.com/tomwright/dasel/v3/model"
	"github.com/tomwright/dasel/v3/selector/ast"
)

func reduceExprExecutor(e ast.ReduceExpr) (expressionExecutor, error) {
	return func(ctx context.Context, options *Options, data *model.Value) (*model.Value, error) {
		ctx = WithExecutorID(ctx, "reduceExpr")
		if !data.IsSlice() {
			return nil, fmt.Errorf("cannot reduce over non-array")
		}

		// Evaluate init expression to get the initial accumulator value.
		acc, err := ExecuteAST(ctx, e.Init, data, options)
		if err != nil {
			return nil, fmt.Errorf("error evaluating reduce init: %w", err)
		}

		// Save and restore the acc variable to avoid polluting the caller's scope.
		prevAcc, hadAcc := options.lookupVar("acc")
		defer func() {
			if hadAcc {
				options.setVar("acc", prevAcc)
			} else {
				options.deleteVar("acc")
			}
		}()

		if err := data.RangeSlice(func(i int, item *model.Value) error {
			restore := withKeyVar(options, model.NewIntValue(int64(i)))
			defer restore()

			// Evaluate the per-element expression against the item.
			elemVal, err := ExecuteAST(ctx, e.Expr, item, options)
			if err != nil {
				return fmt.Errorf("error evaluating reduce expr for element %d: %w", i, err)
			}

			// Set $acc to the current accumulator and $this to the element value.
			options.setVar("acc", acc)
			options.setVar("this", elemVal)

			// Evaluate the update expression.
			newAcc, err := ExecuteAST(ctx, e.Update, elemVal, options)
			if err != nil {
				return fmt.Errorf("error evaluating reduce update for element %d: %w", i, err)
			}
			acc = newAcc
			return nil
		}); err != nil {
			return nil, fmt.Errorf("error ranging over slice: %w", err)
		}

		return acc, nil
	}, nil
}
