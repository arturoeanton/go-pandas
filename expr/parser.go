package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/arturoeanton/go-pandas/errs"
)

// ParseQuery parses a small pandas-like query language into a Predicate:
//
//	age > 30
//	age >= 30 and salary < 2000
//	name == "Ana" or not (age < 18)
//	country in ["AR", "BR"]
func ParseQuery(q string) (Predicate, error) {
	p := &parser{tokens: tokenize(q)}
	pred, err := p.parseOr()
	if err != nil {
		return nil, fmt.Errorf("query %q: %w", q, err)
	}
	if p.pos != len(p.tokens) {
		return nil, fmt.Errorf("query %q: unexpected token %q", q, p.tokens[p.pos])
	}
	return pred, nil
}

func tokenize(q string) []string {
	var tokens []string
	i := 0
	for i < len(q) {
		c := q[i]
		switch {
		case unicode.IsSpace(rune(c)):
			i++
		case c == '"' || c == '\'':
			quote := c
			j := i + 1
			for j < len(q) && q[j] != quote {
				j++
			}
			tokens = append(tokens, q[i:min(j+1, len(q))])
			i = j + 1
		case strings.ContainsRune("()[],+-*/%", rune(c)):
			tokens = append(tokens, string(c))
			i++
		case strings.ContainsRune("<>=!", rune(c)):
			j := i + 1
			if j < len(q) && q[j] == '=' {
				j++
			}
			tokens = append(tokens, q[i:j])
			i = j
		default:
			j := i
			for j < len(q) && !unicode.IsSpace(rune(q[j])) && !strings.ContainsRune("()[],<>=!\"'+-*/%", rune(q[j])) {
				j++
			}
			tokens = append(tokens, q[i:j])
			i = j
		}
	}
	return tokens
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) peek() string {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return ""
}

func (p *parser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *parser) parseOr() (Predicate, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for strings.EqualFold(p.peek(), "or") {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = Or(left, right)
	}
	return left, nil
}

func (p *parser) parseAnd() (Predicate, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for strings.EqualFold(p.peek(), "and") {
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = And(left, right)
	}
	return left, nil
}

func (p *parser) parseUnary() (Predicate, error) {
	if strings.EqualFold(p.peek(), "not") {
		p.next()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return Not(inner), nil
	}
	if p.peek() == "(" {
		// "(" may open a predicate group ("(a > 1) and b") or an
		// arithmetic group ("(a + b) > 1"): try the predicate first and
		// backtrack when what follows the ")" is an operator (v0.10).
		save := p.pos
		p.next()
		inner, err := p.parseOr()
		if err == nil && p.peek() == ")" {
			p.pos++
			if !isBinaryOpToken(p.peek()) {
				return inner, nil
			}
		}
		p.pos = save
	}
	return p.parseComparison()
}

// isBinaryOpToken reports whether a token continues an arithmetic or
// comparison expression.
func isBinaryOpToken(tok string) bool {
	switch tok {
	case "+", "-", "*", "/", "%", "==", "=", "!=", ">", ">=", "<", "<=":
		return true
	}
	return strings.EqualFold(tok, "in")
}

func (p *parser) parseComparison() (Predicate, error) {
	// name.str.contains("x") / startswith / endswith
	if strings.Contains(p.peek(), ".str.") {
		return p.parseStrMethod(p.next())
	}
	left, leftCol, err := p.parseArith()
	if err != nil {
		return nil, err
	}
	next := p.peek()
	// Bare boolean column: `active`, `not active`, `a and active`.
	if leftCol != nil {
		switch {
		case next == "" || next == ")" || strings.EqualFold(next, "and") || strings.EqualFold(next, "or"):
			return leftCol.Eq(true), nil
		}
	}
	// in / not in (v0.10) — require a plain column on the left.
	negIn := strings.EqualFold(next, "not") && strings.EqualFold(p.peekAt(1), "in")
	if negIn || strings.EqualFold(next, "in") {
		if leftCol == nil {
			return nil, fmt.Errorf("%w: 'in' needs a plain column on the left", errs.ErrInvalidOperation)
		}
		if negIn {
			p.next() // not
		}
		p.next() // in
		values, err := p.parseList()
		if err != nil {
			return nil, err
		}
		pred := leftCol.IsIn(values...)
		if negIn {
			pred = Not(pred)
		}
		return pred, nil
	}
	op := p.next()
	switch op {
	case "=":
		op = "=="
	case "==", "!=", ">", ">=", "<", "<=":
	case "":
		return nil, fmt.Errorf("%w: expected comparison operator", errs.ErrInvalidOperation)
	default:
		return nil, fmt.Errorf("%w: unknown operator %q", errs.ErrInvalidOperation, op)
	}
	right, _, err := p.parseArith()
	if err != nil {
		return nil, err
	}
	return comparePred{left: left, right: right, op: op}, nil
}

func (p *parser) peekAt(ahead int) string {
	if p.pos+ahead < len(p.tokens) {
		return p.tokens[p.pos+ahead]
	}
	return ""
}

// parseArith parses an arithmetic expression (v0.10):
//
//	sum    := term (('+'|'-') term)*
//	term   := factor (('*'|'/'|'%') factor)*
//	factor := '-' factor | '(' sum ')' | literal | column
//
// The second return is the ColumnExpr when the whole expression is one
// bare column (used by bare-bool and in/not-in handling).
func (p *parser) parseArith() (Expr, *ColumnExpr, error) {
	left, leftCol, err := p.parseTerm()
	if err != nil {
		return nil, nil, err
	}
	for p.peek() == "+" || p.peek() == "-" {
		op := p.next()
		right, _, err := p.parseTerm()
		if err != nil {
			return nil, nil, err
		}
		left, leftCol = binaryExpr{left: left, right: right, op: op}, nil
	}
	return left, leftCol, nil
}

func (p *parser) parseTerm() (Expr, *ColumnExpr, error) {
	left, leftCol, err := p.parseFactor()
	if err != nil {
		return nil, nil, err
	}
	for p.peek() == "*" || p.peek() == "/" || p.peek() == "%" {
		op := p.next()
		right, _, err := p.parseFactor()
		if err != nil {
			return nil, nil, err
		}
		left, leftCol = binaryExpr{left: left, right: right, op: op}, nil
	}
	return left, leftCol, nil
}

func (p *parser) parseFactor() (Expr, *ColumnExpr, error) {
	tok := p.peek()
	switch {
	case tok == "":
		return nil, nil, fmt.Errorf("%w: expected value or column", errs.ErrInvalidOperation)
	case tok == "-":
		p.next()
		inner, _, err := p.parseFactor()
		if err != nil {
			return nil, nil, err
		}
		return binaryExpr{left: Lit(0), right: inner, op: "-"}, nil, nil
	case tok == "(":
		p.next()
		inner, _, err := p.parseArith()
		if err != nil {
			return nil, nil, err
		}
		if p.next() != ")" {
			return nil, nil, fmt.Errorf("%w: expected ')'", errs.ErrInvalidOperation)
		}
		return inner, nil, nil
	}
	p.next()
	// Literals first: quoted strings, numbers, bools, NA. Identifiers
	// that are not literal keywords fail parseValue and become columns.
	if v, err := parseValue(tok); err == nil {
		return Lit(v), nil, nil
	}
	if !isIdentifier(tok) {
		return nil, nil, fmt.Errorf("%w: cannot parse %q", errs.ErrInvalidOperation, tok)
	}
	col := Col(tok)
	return col, &col, nil
}

// isIdentifier reports whether a token looks like a column name.
func isIdentifier(tok string) bool {
	if tok == "" {
		return false
	}
	for i, r := range tok {
		if unicode.IsLetter(r) || r == '_' || (i > 0 && (unicode.IsDigit(r) || r == '.')) {
			continue
		}
		return false
	}
	return true
}

// parseStrMethod handles `col.str.method("arg")` calls in queries.
func (p *parser) parseStrMethod(tok string) (Predicate, error) {
	parts := strings.SplitN(tok, ".str.", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: malformed str accessor %q", errs.ErrInvalidOperation, tok)
	}
	col, method := Col(parts[0]), parts[1]
	if p.next() != "(" {
		return nil, fmt.Errorf("%w: expected '(' after .str.%s", errs.ErrInvalidOperation, method)
	}
	argTok := p.next()
	arg, err := parseValue(argTok)
	if err != nil {
		return nil, err
	}
	argStr, ok := arg.(string)
	if !ok {
		return nil, fmt.Errorf("%w: .str.%s expects a string argument", errs.ErrInvalidOperation, method)
	}
	if p.next() != ")" {
		return nil, fmt.Errorf("%w: expected ')' after .str.%s(...)", errs.ErrInvalidOperation, method)
	}
	switch method {
	case "contains":
		return col.Contains(argStr), nil
	case "startswith":
		return col.StartsWith(argStr), nil
	case "endswith":
		return col.EndsWith(argStr), nil
	}
	return nil, errs.NotImplemented("query .str." + method)
}

func (p *parser) parseList() ([]any, error) {
	if p.next() != "[" {
		return nil, fmt.Errorf("%w: expected '[' after 'in'", errs.ErrInvalidOperation)
	}
	var out []any
	for {
		t := p.next()
		if t == "]" {
			return out, nil
		}
		if t == "," {
			continue
		}
		if t == "" {
			return nil, fmt.Errorf("%w: unterminated list", errs.ErrInvalidOperation)
		}
		v, err := parseValue(t)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

func parseValue(tok string) (any, error) {
	if tok == "" {
		return nil, fmt.Errorf("%w: expected value", errs.ErrInvalidOperation)
	}
	if len(tok) >= 2 && (tok[0] == '"' || tok[0] == '\'') {
		return tok[1 : len(tok)-1], nil
	}
	switch strings.ToLower(tok) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null", "none", "na", "nan":
		return nil, nil
	}
	if i, err := strconv.ParseInt(tok, 10, 64); err == nil {
		return int(i), nil
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("%w: cannot parse value %q", errs.ErrInvalidOperation, tok)
}
