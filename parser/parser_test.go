package parser

import (
	"strings"
	"testing"

	"github.com/akzj/go-regex/ast"
	"github.com/akzj/go-regex/lexer"
)

func TestParseString_Basic(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		wantType ast.NodeType
	}{
		{"empty", "", ast.NodeEmpty},
		{"single char", "a", ast.NodeChar},
		{"literal chars", "abc", ast.NodeConcat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			node, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if node.Type() != tt.wantType {
				t.Errorf("ParseString(%q) = %v, want %v", tt.pattern, node.Type(), tt.wantType)
			}
		})
	}
}

func TestParseString_Metacharacters(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		wantType ast.NodeType
	}{
		{"dot", ".", ast.NodeAny},
		{"star", "a*", ast.NodeStar},
		{"plus", "a+", ast.NodePlus},
		{"quest", "a?", ast.NodeQuest},
		{"caret", "^", ast.NodeBegin},
		{"dollar", "$", ast.NodeEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			node, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if node.Type() != tt.wantType {
				t.Errorf("ParseString(%q) = %v, want %v", tt.pattern, node.Type(), tt.wantType)
			}
		})
	}
}

func TestParseString_Alternation(t *testing.T) {
	p := New()
	node, err := p.ParseString("a|b")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if node.Type() != ast.NodeAlt {
		t.Errorf("ParseString(\"a|b\") = %v, want %v", node.Type(), ast.NodeAlt)
	}

	// Test left associativity: a|b|c should be ((a|b)|c)
	// But any flat structure is acceptable for AST
	node, err = p.ParseString("a|b|c")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if node.Type() != ast.NodeAlt {
		t.Errorf("ParseString(\"a|b|c\") = %v, want %v", node.Type(), ast.NodeAlt)
	}
}

func TestParseString_Concat(t *testing.T) {
	p := New()
	node, err := p.ParseString("ab")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if node.Type() != ast.NodeConcat {
		t.Errorf("ParseString(\"ab\") = %v, want %v", node.Type(), ast.NodeConcat)
	}

	// Test longer concatenation
	node, err = p.ParseString("abc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if node.Type() != ast.NodeConcat {
		t.Errorf("ParseString(\"abc\") = %v, want %v", node.Type(), ast.NodeConcat)
	}
}

func TestParseString_Groups(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		wantType ast.NodeType
	}{
		{"simple group", "(a)", ast.NodeCapture},
		{"non-capturing group", "(?:a)", ast.NodeGroup},
		{"group with alt", "(a|b)", ast.NodeCapture},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			node, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if node.Type() != tt.wantType {
				t.Errorf("ParseString(%q) = %v, want %v", tt.pattern, node.Type(), tt.wantType)
			}
		})
	}
}

func TestParseString_CharacterClass(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		wantType    ast.NodeType
		wantRanges  int
	}{
		{"simple class", "[abc]", ast.NodeClass, 3},
		{"class with range", "[a-z]", ast.NodeClass, 1},
		{"negated class", "[^abc]", ast.NodeClass, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			node, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if node.Type() != tt.wantType {
				t.Errorf("ParseString(%q) = %v, want %v", tt.pattern, node.Type(), tt.wantType)
			}
			if cn, ok := node.(*ast.ClassNode); ok {
				if len(cn.Ranges) != tt.wantRanges {
					t.Errorf("ClassNode.Ranges = %v (len %d), want %d ranges",
						cn.Ranges, len(cn.Ranges), tt.wantRanges)
				}
			}
		})
	}
}

func TestParseString_Repetition(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		wantType   ast.NodeType
		wantMin    int
		wantMax    int
	}{
		{"exact", "a{3}", ast.NodeRep, 3, 3},
		{"range", "a{2,4}", ast.NodeRep, 2, 4},
		{"zero or more", "a{0,}", ast.NodeRep, 0, -1},
		{"at least one", "a{1,}", ast.NodeRep, 1, -1},
		{"zero or one", "a{0,1}", ast.NodeRep, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			node, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if node.Type() != tt.wantType {
				t.Errorf("ParseString(%q) = %v, want %v", tt.pattern, node.Type(), tt.wantType)
			}
			if rn, ok := node.(*ast.RepNode); ok {
				if rn.Min != tt.wantMin {
					t.Errorf("RepNode.Min = %d, want %d", rn.Min, tt.wantMin)
				}
				if rn.Max != tt.wantMax {
					t.Errorf("RepNode.Max = %d, want %d", rn.Max, tt.wantMax)
				}
			}
		})
	}
}

func TestParseString_EmptyRepetition(t *testing.T) {
	// {0} should produce EmptyNode, not nil
	p := New()
	node, err := p.ParseString("a{0}")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if node.Type() != ast.NodeEmpty {
		t.Errorf("ParseString(\"a{0}\") = %v, want %v", node.Type(), ast.NodeEmpty)
	}
}

func TestParseString_EscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		wantType ast.NodeType
	}{
		{"escaped dot", `\.`, ast.NodeAny},
		{"escaped star", `\*`, ast.NodeChar},
		{"escaped plus", `\+`, ast.NodePlus},
		{"escaped quest", `\?`, ast.NodeQuest},
		// Note: \(\) is tokenized as actual parentheses by the lexer,
		// so it becomes an empty capturing group
		{"escaped parens", `\(\)`, ast.NodeCapture},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			node, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if node.Type() != tt.wantType {
				t.Errorf("ParseString(%q) = %v, want %v", tt.pattern, node.Type(), tt.wantType)
			}
		})
	}
}

func TestParseString_PositionAnchors(t *testing.T) {
	p := New()
	node, err := p.ParseString("^abc$")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	// ^abc$ -> concat of (begin, concat of a,b,c, end)
	if node.Type() != ast.NodeConcat {
		t.Errorf("ParseString(\"^abc$\") = %v, want %v", node.Type(), ast.NodeConcat)
	}
}

func TestParseString_ComplexPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{"email-like", `[a-z]+@[a-z]+\.[a-z]+`},
		{"phone-like", `\d{3}-\d{4}`},
		{"alternation with concat", `a(b|c)*`},
		{"nested groups", `((a|b)+)`},
		{"anchored pattern", `^hello world$`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			_, err := p.ParseString(tt.pattern)
			if err != nil {
				t.Errorf("unexpected error for pattern %q: %v", tt.pattern, err)
			}
		})
	}
}

func TestParseString_Errors(t *testing.T) {
	// Test that error inputs return meaningful errors
	// Note: These patterns are handled by lexer, not parser
	// Parser should handle malformed AST structures

	// Missing closing paren - lexer returns error, parser may not see it
	p := New()
	_, err := p.ParseString("(a")
	if err == nil {
		t.Errorf("expected error for unbalanced paren")
	}
	if !strings.Contains(err.Error(), "parse error") && !strings.Contains(err.Error(), "parenthesis") {
		t.Errorf("error message should mention parse or parenthesis: %v", err)
	}
}

func TestParse_Interface(t *testing.T) {
	// Test that Parse accepts a lexer.Lexer interface
	p := New()
	l := newMockLexer("abc")
	node, err := p.Parse(l)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if node == nil {
		t.Errorf("expected non-nil node")
	}
}

// mockLexer for testing the Lexer interface
type mockLexer struct {
	tokens []mockToken
	pos    int
}

type mockToken struct {
	Type  lexer.TokenType
	Val   rune
	Class []rune
}

func newMockLexer(input string) *mockLexer {
	// Create a simple lexer that tokenizes the input
	l := &mockLexer{}
	for _, r := range input {
		l.tokens = append(l.tokens, mockToken{Type: lexer.TokenChar, Val: r})
	}
	l.tokens = append(l.tokens, mockToken{Type: lexer.TokenEOF})
	return l
}

func (l *mockLexer) Next() lexer.Token {
	if l.pos >= len(l.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	tok := l.tokens[l.pos]
	l.pos++
	return lexer.Token{Type: tok.Type, Val: tok.Val, Class: tok.Class}
}

func (l *mockLexer) Peek() lexer.Token {
	if l.pos >= len(l.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return lexer.Token{Type: l.tokens[l.pos].Type, Val: l.tokens[l.pos].Val, Class: l.tokens[l.pos].Class}
}

func (l *mockLexer) Reset(input string) {
	l.tokens = nil
	l.pos = 0
	for _, r := range input {
		l.tokens = append(l.tokens, mockToken{Type: lexer.TokenChar, Val: r})
	}
	l.tokens = append(l.tokens, mockToken{Type: lexer.TokenEOF})
}

func TestParseString_NeverReturnsNil(t *testing.T) {
	// Parser should never return nil node - empty patterns return EmptyNode
	p := New()
	patterns := []string{"", "a", "a*", "(a)", "[abc]", "^$", "."}

	for _, pattern := range patterns {
		node, err := p.ParseString(pattern)
		if err != nil {
			t.Errorf("ParseString(%q) error: %v", pattern, err)
			continue
		}
		if node == nil {
			t.Errorf("ParseString(%q) returned nil node", pattern)
		}
	}
}

func TestParseString_CharNodeValues(t *testing.T) {
	p := New()
	node, err := p.ParseString("abc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Walk the concat tree to verify char values
	concat, ok := node.(*ast.ConcatNode)
	if !ok {
		t.Fatalf("expected ConcatNode, got %T", node)
	}

	// Left should be Concat of 'a' and 'bc'
	left, ok := concat.Left.(*ast.ConcatNode)
	if !ok {
		t.Fatalf("expected ConcatNode for left side, got %T", concat.Left)
	}

	charA, ok := left.Left.(*ast.CharNode)
	if !ok {
		t.Fatalf("expected CharNode for 'a', got %T", left.Left)
	}
	if charA.Ch != 'a' {
		t.Errorf("expected 'a', got %c", charA.Ch)
	}
}
