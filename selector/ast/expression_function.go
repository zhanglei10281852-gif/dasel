package ast

// FunctionExpr represents a user-defined function declared with def.
//
// The function is not an executable expression on its own: it is collected
// within a ProgramExpr and invoked through a CallExpr.
type FunctionExpr struct {
	// Name is the name the function can be called with.
	Name string
	// Params holds the parameter variable names (without the leading $).
	Params []string
	// Body is evaluated when the function is called. It executes against
	// the current data at the call site and returns the result of the
	// last expression in the body.
	Body Expr
}

// ProgramExpr wraps a query that starts with one or more function declarations.
//
// The declarations are scoped to the query and must not be registered on the
// host's native function collection.
type ProgramExpr struct {
	// Functions holds the function declarations in the order they were declared.
	Functions []FunctionExpr
	// Expr is the main query expression executed after the declarations.
	Expr Expr
}

func (ProgramExpr) expr() {}
