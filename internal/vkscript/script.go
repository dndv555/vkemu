package vkscript

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type API func(method string, args map[string]any) (any, error)

func Eval(code string, api API) (any, error) {
	tokens, err := lex(code)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens, api: api, vars: map[string]any{}}
	return p.run()
}

func Stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "1"
		}
		return "0"
	case float64:
		return formatNumber(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case []any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			parts = append(parts, Stringify(item))
		}
		return strings.Join(parts, ",")
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenIdent
	tokenNumber
	tokenString
	tokenPunct
)

type token struct {
	kind tokenKind
	val  string
	num  float64
}

func lex(src string) ([]token, error) {
	tokens := make([]token, 0, len(src)/3)
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '"' || c == '\'':
			quote := c
			i++
			var sb strings.Builder
			for i < len(src) && src[i] != quote {
				if src[i] == '\\' && i+1 < len(src) {
					i++
					switch src[i] {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					case 'r':
						sb.WriteByte('\r')
					default:
						sb.WriteByte(src[i])
					}
					i++
					continue
				}
				sb.WriteByte(src[i])
				i++
			}
			if i >= len(src) {
				return nil, fmt.Errorf("vkscript: unterminated string")
			}
			i++
			tokens = append(tokens, token{kind: tokenString, val: sb.String()})
		case c >= '0' && c <= '9':
			start := i
			for i < len(src) && (src[i] >= '0' && src[i] <= '9' || src[i] == '.') {
				i++
			}
			text := src[start:i]
			value, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return nil, fmt.Errorf("vkscript: bad number %q", text)
			}
			tokens = append(tokens, token{kind: tokenNumber, val: text, num: value})
		case isIdentStart(c):
			start := i
			for i < len(src) && isIdentPart(src[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokenIdent, val: src[start:i]})
		default:
			tokens = append(tokens, token{kind: tokenPunct, val: string(c)})
			i++
		}
	}
	tokens = append(tokens, token{kind: tokenEOF})
	return tokens, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

type parser struct {
	tokens []token
	pos    int
	api    API
	vars   map[string]any
}

func (p *parser) run() (any, error) {
	for {
		t := p.peek()
		if t.kind == tokenEOF {
			return nil, nil
		}
		if p.isPunct(";") {
			p.next()
			continue
		}
		if t.kind == tokenIdent && t.val == "var" {
			p.next()
			name, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			p.expectPunct("=")
			value, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			p.expectPunct(";")
			p.vars[name] = value
			continue
		}
		if t.kind == tokenIdent && t.val == "return" {
			p.next()
			value, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if p.isPunct(";") {
				p.next()
			}
			return value, nil
		}
		if _, err := p.parseExpr(); err != nil {
			return nil, err
		}
		if p.isPunct(";") {
			p.next()
		}
	}
}

func (p *parser) peek() token {
	return p.tokens[p.pos]
}

func (p *parser) next() token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *parser) isPunct(val string) bool {
	t := p.peek()
	return t.kind == tokenPunct && t.val == val
}

func (p *parser) acceptPunct(val string) bool {
	if p.isPunct(val) {
		p.next()
		return true
	}
	return false
}

func (p *parser) expectPunct(val string) error {
	if !p.acceptPunct(val) {
		return fmt.Errorf("vkscript: expected %q", val)
	}
	return nil
}

func (p *parser) expectIdent() (string, error) {
	t := p.peek()
	if t.kind != tokenIdent {
		return "", fmt.Errorf("vkscript: expected identifier")
	}
	p.next()
	return t.val, nil
}

func (p *parser) parseExpr() (any, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.isPunct("+") {
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = add(left, right)
	}
	return left, nil
}

func (p *parser) parseUnary() (any, error) {
	if p.acceptPunct("-") {
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		if f, ok := asFloat(value); ok {
			return -f, nil
		}
		return nil, fmt.Errorf("vkscript: bad unary minus")
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (any, error) {
	value, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case p.isPunct("["):
			p.next()
			index, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if err := p.expectPunct("]"); err != nil {
				return nil, err
			}
			value = indexOf(value, index)
		case p.isPunct("."):
			p.next()
			name, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			value = member(value, name)
		case p.isPunct("@"):
			p.next()
			if err := p.expectPunct("."); err != nil {
				return nil, err
			}
			name, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			value = project(value, name)
		default:
			return value, nil
		}
	}
}

func (p *parser) parseAtom() (any, error) {
	t := p.peek()
	switch {
	case t.kind == tokenNumber:
		p.next()
		return t.num, nil
	case t.kind == tokenString:
		p.next()
		return t.val, nil
	case t.kind == tokenPunct && t.val == "{":
		return p.parseObject()
	case t.kind == tokenPunct && t.val == "[":
		return p.parseArray()
	case t.kind == tokenIdent:
		p.next()
		switch t.val {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null":
			return nil, nil
		}
		if t.val == "API" {
			return p.parseAPICall()
		}
		if value, ok := p.vars[t.val]; ok {
			return value, nil
		}
		return nil, nil
	}
	return nil, fmt.Errorf("vkscript: unexpected token %q", t.val)
}

func (p *parser) parseAPICall() (any, error) {
	path := make([]string, 0, 2)
	for p.isPunct(".") {
		p.next()
		name, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		path = append(path, name)
	}
	if len(path) == 0 {
		return nil, fmt.Errorf("vkscript: empty api path")
	}
	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}
	value, err := p.api(strings.Join(path, "."), args)
	if err != nil {
		return nil, err
	}
	return normalize(value), nil
}

func (p *parser) parseArgs() (map[string]any, error) {
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}
	args := map[string]any{}
	if p.acceptPunct(")") {
		return args, nil
	}
	for {
		if p.isPunct("{") {
			obj, err := p.parseObject()
			if err != nil {
				return nil, err
			}
			for k, v := range obj {
				args[k] = v
			}
		} else {
			if _, err := p.parseExpr(); err != nil {
				return nil, err
			}
		}
		if p.acceptPunct(",") {
			continue
		}
		break
	}
	if err := p.expectPunct(")"); err != nil {
		return nil, err
	}
	return args, nil
}

func (p *parser) parseObject() (map[string]any, error) {
	if err := p.expectPunct("{"); err != nil {
		return nil, err
	}
	obj := map[string]any{}
	if p.acceptPunct("}") {
		return obj, nil
	}
	for {
		t := p.peek()
		var key string
		switch t.kind {
		case tokenIdent, tokenString, tokenNumber:
			key = t.val
			p.next()
		default:
			return nil, fmt.Errorf("vkscript: bad object key %q", t.val)
		}
		if err := p.expectPunct(":"); err != nil {
			return nil, err
		}
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		obj[key] = value
		if p.acceptPunct(",") {
			continue
		}
		break
	}
	if err := p.expectPunct("}"); err != nil {
		return nil, err
	}
	return obj, nil
}

func (p *parser) parseArray() (any, error) {
	if err := p.expectPunct("["); err != nil {
		return nil, err
	}
	items := []any{}
	if p.acceptPunct("]") {
		return items, nil
	}
	for {
		value, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		items = append(items, value)
		if p.acceptPunct(",") {
			continue
		}
		break
	}
	if err := p.expectPunct("]"); err != nil {
		return nil, err
	}
	return items, nil
}

func normalize(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return v
	}
	return out
}

func add(a, b any) any {
	af, aok := asFloat(a)
	bf, bok := asFloat(b)
	if aok && bok {
		return af + bf
	}
	return Stringify(a) + Stringify(b)
}

func member(v any, name string) any {
	if obj, ok := v.(map[string]any); ok {
		return obj[name]
	}
	return nil
}

func indexOf(v, index any) any {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	f, ok := asFloat(index)
	if !ok {
		return nil
	}
	i := int(f)
	if i < 0 || i >= len(items) {
		return nil
	}
	return items[i]
}

func project(v any, name string) any {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		value, ok := obj[name]
		if !ok || value == nil {
			continue
		}
		out = append(out, value)
	}
	return out
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

func formatNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
