package parser

import (
	"fmt"

	"github.com/tomwright/dasel/v3/selector/ast"
	"github.com/tomwright/dasel/v3/selector/lexer"
)

// isDefDeclarationStart reports whether the current token starts a function
// declaration.
//
// "def" is a contextual keyword: it is only treated as a declaration when the
// full `def name(` shape is present. A bare "def" symbol anywhere else (for
// example a data field named "def") continues to parse as a property.
func (p *Parser) isDefDeclarationStart() bool {
	return p.hasToken() &&
		p.current().Kind == lexer.Symbol &&
		p.current().Value == "def" &&
		p.peek().Kind == lexer.Symbol &&
		p.peekN(2).Kind == lexer.OpenParen
}

// parseFunctionDefs parses any leading function declarations.
//
// It returns nil when the query does not start with a declaration, leaving
// the parser position untouched so the query is parsed as before.
func (p *Parser) parseFunctionDefs() ([]ast.FunctionExpr, error) {
	var defs []ast.FunctionExpr
	defined := make(map[string]bool)

	for p.isDefDeclarationStart() {
		def, err := p.parseFunctionDef()
		if err != nil {
			return nil, err
		}
		if defined[def.Name] {
			return nil, fmt.Errorf("function %q is defined more than once", def.Name)
		}
		defined[def.Name] = true
		defs = append(defs, def)
	}

	return defs, nil
}

func (p *Parser) parseFunctionDef() (ast.FunctionExpr, error) {
	var res ast.FunctionExpr

	// Consume "def".
	p.advance()

	// Function name.
	if err := p.expect(lexer.Symbol); err != nil {
		return res, err
	}
	res.Name = p.current().Value
	p.advance()

	// Open parenthesis.
	if err := p.expect(lexer.OpenParen); err != nil {
		return res, err
	}
	p.advance()

	// Parameters.
	params := make([]string, 0)
	seenParams := make(map[string]bool)
	for !p.current().IsKind(lexer.CloseParen) {
		if err := p.expect(lexer.Variable); err != nil {
			return res, fmt.Errorf("function %q parameters must be variables prefixed with $: %w", res.Name, err)
		}
		paramName := p.current().Value
		if seenParams[paramName] {
			return res, fmt.Errorf("duplicate parameter %q in function %q", paramName, res.Name)
		}
		seenParams[paramName] = true
		params = append(params, paramName)
		p.advance()

		if p.current().IsKind(lexer.Comma) {
			p.advance()
			continue
		}
		if err := p.expect(lexer.CloseParen); err != nil {
			return res, err
		}
	}
	res.Params = params
	p.advance() // consume close parenthesis

	// Body.
	if err := p.expect(lexer.OpenCurly); err != nil {
		return res, err
	}
	p.advance()

	body, err := p.parseExpressions(
		[]lexer.TokenKind{lexer.CloseCurly},
		[]lexer.TokenKind{lexer.Semicolon},
		true,
		bpDefault,
		true,
	)
	if err != nil {
		return res, err
	}
	if body == nil {
		return res, fmt.Errorf("function %q body must contain at least one expression", res.Name)
	}
	res.Body = body

	// Declarations are terminated with a semicolon.
	if err := p.expect(lexer.Semicolon); err != nil {
		return res, fmt.Errorf("function %q declaration must end with a semicolon: %w", res.Name, err)
	}
	p.advance()

	return res, nil
}
