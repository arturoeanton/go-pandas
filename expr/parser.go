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
		case strings.ContainsRune("()[],", rune(c)):
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
			for j < len(q) && !unicode.IsSpace(rune(q[j])) && !strings.ContainsRune("()[],<>=!\"'", rune(q[j])) {
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
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.next() != ")" {
			return nil, fmt.Errorf("%w: expected ')'", errs.ErrInvalidOperation)
		}
		return inner, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (Predicate, error) {
	colTok := p.next()
	if colTok == "" {
		return nil, fmt.Errorf("%w: expected column name", errs.ErrInvalidOperation)
	}
	col := Col(colTok)
	op := p.next()
	if strings.EqualFold(op, "in") {
		values, err := p.parseList()
		if err != nil {
			return nil, err
		}
		return col.IsIn(values...), nil
	}
	valTok := p.next()
	val, err := parseValue(valTok)
	if err != nil {
		return nil, err
	}
	switch op {
	case "==", "=":
		return col.Eq(val), nil
	case "!=":
		return col.Ne(val), nil
	case ">":
		return col.Gt(val), nil
	case ">=":
		return col.Ge(val), nil
	case "<":
		return col.Lt(val), nil
	case "<=":
		return col.Le(val), nil
	}
	return nil, fmt.Errorf("%w: unknown operator %q", errs.ErrInvalidOperation, op)
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
