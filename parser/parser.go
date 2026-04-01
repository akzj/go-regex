package parser

import (
	"fmt"
	"strings"

	"github.com/akzj/go-regex/ast"
	"github.com/akzj/go-regex/lexer"
)

// Parser interface definition
type Parser interface {
	Parse(l lexer.Lexer) (ast.Node, error)
	ParseString(pattern string) (ast.Node, error)
}

// parser implements the Parser interface
type parser struct {
	l      lexer.Lexer
	pos    int
	errors []string
}

// New creates a standard parser
func New() Parser {
	return &parser{}
}

// Parse parses a regex pattern from a lexer
func (p *parser) Parse(l lexer.Lexer) (ast.Node, error) {
	p.l = l
	p.pos = 0
	p.errors = nil

	node := p.parseExpr()
	if node == nil {
		node = &ast.EmptyNode{}
	}

	// Check for trailing errors
	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse error: %s", strings.Join(p.errors, "; "))
	}

	// Check that we consumed all tokens
	tok := p.l.Peek()
	if tok.Type != lexer.TokenEOF {
		return nil, fmt.Errorf("parse error: unexpected token %v at position %d", tok.Type, tok.Pos)
	}

	return node, nil
}

// ParseString parses a regex pattern from a string
func (p *parser) ParseString(pattern string) (ast.Node, error) {
	if pattern == "" {
		return &ast.EmptyNode{}, nil
	}
	l := lexer.New()
	l.Reset(pattern)
	return p.Parse(l)
}

// parseExpr parses the top-level expression (alternation)
func (p *parser) parseExpr() ast.Node {
	node := p.parseConcat()

	// Handle alternation
	tok := p.l.Peek()
	if tok.Type == lexer.TokenBar {
		p.l.Next() // consume '|'
		right := p.parseExpr()
		if right == nil {
			right = &ast.EmptyNode{}
		}
		if node == nil {
			node = &ast.EmptyNode{}
		}
		node = &ast.AltNode{Left: node, Right: right}
	}

	return node
}

// parseConcat parses concatenation of terms
func (p *parser) parseConcat() ast.Node {
	var terms []ast.Node

	for {
		tok := p.l.Peek()
		// RParen terminates concatenation (for group parsing)
		if tok.Type == lexer.TokenEOF || tok.Type == lexer.TokenBar || tok.Type == lexer.TokenRParen {
			break
		}

		node := p.parseTerm()
		if node == nil {
			break
		}
		terms = append(terms, node)
	}

	if len(terms) == 0 {
		return nil
	}
	if len(terms) == 1 {
		return terms[0]
	}

	// Build left-associative concatenation tree
	result := terms[0]
	for i := 1; i < len(terms); i++ {
		result = &ast.ConcatNode{Left: result, Right: terms[i]}
	}
	return result
}

// parseTerm parses a single term (atom with optional quantifier)
func (p *parser) parseTerm() ast.Node {
	node := p.parseAtom()
	if node == nil {
		return nil
	}

	// Check for quantifiers
	tok := p.l.Peek()
	switch tok.Type {
	case lexer.TokenStar:
		p.l.Next()
		node = &ast.StarNode{Node: node}
	case lexer.TokenPlus:
		p.l.Next()
		node = &ast.PlusNode{Node: node}
	case lexer.TokenQuest:
		p.l.Next()
		node = &ast.QuestNode{Node: node}
	case lexer.TokenLBrace:
		node = p.parseRepetition(node)
	}

	return node
}

// parseAtom parses an atomic expression
// Key: We peek first, then only consume if we decide to handle the token
func (p *parser) parseAtom() ast.Node {
	tok := p.l.Peek()

	switch tok.Type {
	case lexer.TokenEOF, lexer.TokenBar, lexer.TokenRParen:
		// These are terminators, don't consume
		return nil

	case lexer.TokenChar:
		p.l.Next() // consume
		return &ast.CharNode{Ch: tok.Val}

	case lexer.TokenDot:
		p.l.Next()
		return &ast.AnyNode{}

	case lexer.TokenStar:
		p.l.Next()
		return &ast.StarNode{Node: &ast.EmptyNode{}}

	case lexer.TokenPlus:
		p.l.Next()
		return &ast.PlusNode{Node: &ast.EmptyNode{}}

	case lexer.TokenQuest:
		p.l.Next()
		return &ast.QuestNode{Node: &ast.EmptyNode{}}

	case lexer.TokenCaret:
		p.l.Next()
		return &ast.BeginNode{}

	case lexer.TokenDollar:
		p.l.Next()
		return &ast.EndNode{}

	case lexer.TokenLParen:
		p.l.Next()
		return p.parseGroup()

	case lexer.TokenLBracket:
		p.l.Next()
		return p.parseClass(tok)

	case lexer.TokenDash:
		// Standalone dash outside character class - treat as literal
		p.l.Next()
		return &ast.CharNode{Ch: '-'}

	default:
		p.l.Next() // consume to avoid infinite loop
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %v at position %d", tok.Type, tok.Pos))
		return nil
	}
}

// parseGroup parses a group expression
func (p *parser) parseGroup() ast.Node {
	// Check for non-capturing group (?:...)
	tok := p.l.Peek()
	if tok.Type == lexer.TokenQuest {
		p.l.Next() // consume '?'
		tok = p.l.Peek()
		if tok.Type == lexer.TokenChar && tok.Val == ':' {
			p.l.Next() // consume ':'
			node := p.parseExpr()
			if node == nil {
				node = &ast.EmptyNode{}
			}
			// Expect closing paren
			if p.l.Peek().Type != lexer.TokenRParen {
				p.errors = append(p.errors, "missing closing parenthesis")
			} else {
				p.l.Next() // consume ')'
			}
			return &ast.GroupNode{Node: node}
		}
		// Unknown (?...) variant - put '?' back by pushing it into expr
		// For simplicity, treat as capturing group with '?' as content
	}

	// Regular capturing group
	node := p.parseExpr()
	if node == nil {
		node = &ast.EmptyNode{}
	}

	// Expect closing paren
	if p.l.Peek().Type != lexer.TokenRParen {
		p.errors = append(p.errors, "missing closing parenthesis")
	} else {
		p.l.Next() // consume ')'
	}

	return &ast.CaptureNode{Node: node}
}

// parseClass parses a character class from TokenLBracket
func (p *parser) parseClass(openTok lexer.Token) ast.Node {
	class := openTok.Class
	negated := false

	// Check if it's a negated class (first char is '^')
	if len(class) > 0 && class[0] == '^' {
		class = class[1:]
		negated = true
	}

	// Parse the ranges from the encoded format
	ranges := p.parseCharRanges(class)

	return &ast.ClassNode{Ranges: ranges, Negated: negated}
}

// parseCharRanges parses the encoded character class format into CharRanges
// Format from lexer: [a, '-', z] for a-z range
//                    [a, b, c] for individual chars
func (p *parser) parseCharRanges(class []rune) []ast.CharRange {
	var ranges []ast.CharRange
	i := 0

	for i < len(class) {
		ch := class[i]

		// Check if this is a range marker
		if ch == '-' && i+1 < len(class) && i > 0 {
			// This is a range marker: previous char is lo, next char is hi
			hi := class[i+1]
			// Update the last range with hi
			if len(ranges) > 0 {
				ranges[len(ranges)-1].Hi = hi
			}
			i += 2
			continue
		}

		// Single character or start of range
		lo := ch
		hi := ch

		// Check if next char is '-' indicating a range
		if i+2 < len(class) && class[i+1] == '-' {
			hi = class[i+2]
			i += 3 // skip lo, '-', hi
		} else {
			i++
		}

		ranges = append(ranges, ast.CharRange{Lo: lo, Hi: hi})
	}

	return ranges
}

// parseRepetition parses {n} or {n,m} repetition
func (p *parser) parseRepetition(child ast.Node) ast.Node {
	p.l.Next() // consume TokenLBrace

	// Collect digits before optional comma
	var num1 []rune
	for {
		tok := p.l.Peek()
		if tok.Type != lexer.TokenChar {
			break
		}
		if tok.Val < '0' || tok.Val > '9' {
			break
		}
		p.l.Next()
		num1 = append(num1, tok.Val)
	}

	if len(num1) == 0 {
		p.errors = append(p.errors, "expected number in repetition")
		// Skip to closing brace
		for p.l.Peek().Type != lexer.TokenRBrace && p.l.Peek().Type != lexer.TokenEOF {
			p.l.Next()
		}
		if p.l.Peek().Type == lexer.TokenRBrace {
			p.l.Next()
		}
		return child
	}

	min := 0
	for _, r := range num1 {
		min = min*10 + int(r-'0')
	}

	max := min

	// Check for comma (TokenChar with Val=',')
	tok := p.l.Peek()
	if tok.Type == lexer.TokenChar && tok.Val == ',' {
		p.l.Next() // consume ','
		
		// Parse optional second number
		var num2 []rune
		for {
			tok := p.l.Peek()
			if tok.Type != lexer.TokenChar {
				break
			}
			if tok.Val < '0' || tok.Val > '9' {
				break
			}
			p.l.Next()
			num2 = append(num2, tok.Val)
		}

		if len(num2) == 0 {
			// {n,} means n or more (max = -1)
			max = -1
		} else {
			max = 0
			for _, r := range num2 {
				max = max*10 + int(r-'0')
			}
		}
	}

	// Expect closing brace
	if p.l.Peek().Type != lexer.TokenRBrace {
		p.errors = append(p.errors, "expected closing brace")
	} else {
		p.l.Next() // consume TokenRBrace
	}

	// Handle {0} as empty
	if min == 0 && max == 0 {
		return &ast.EmptyNode{}
	}

	return &ast.RepNode{Child: child, Min: min, Max: max}
}
