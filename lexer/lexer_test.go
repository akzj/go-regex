package lexer

import (
	"testing"
)

func TestTokenize_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "empty",
			input:    "",
			expected: []TokenType{TokenEOF},
		},
		{
			name:     "single char",
			input:    "a",
			expected: []TokenType{TokenChar, TokenEOF},
		},
		{
			name:     "literal chars",
			input:    "abc",
			expected: []TokenType{TokenChar, TokenChar, TokenChar, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(tokens) != len(tt.expected) {
				t.Errorf("Tokenize(%q) = %v, expected %v tokens", tt.input, tokens, tt.expected)
				return
			}
			for i, expected := range tt.expected {
				if tokens[i].Type != expected {
					t.Errorf("Token[%d] = %v, want %v", i, tokens[i].Type, expected)
				}
			}
		})
	}
}

func TestTokenize_Metacharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{"dot", ".", []TokenType{TokenDot, TokenEOF}},
		{"star", "*", []TokenType{TokenStar, TokenEOF}},
		{"plus", "+", []TokenType{TokenPlus, TokenEOF}},
		{"quest", "?", []TokenType{TokenQuest, TokenEOF}},
		{"bar", "|", []TokenType{TokenBar, TokenEOF}},
		{"lparen", "(", []TokenType{TokenLParen, TokenEOF}},
		{"rparen", ")", []TokenType{TokenRParen, TokenEOF}},
		{"lbracket", "[", []TokenType{TokenLBracket, TokenEOF}},
		{"rbracket", "]", []TokenType{TokenRBracket, TokenEOF}},
		{"lbrace", "{", []TokenType{TokenLBrace, TokenEOF}},
		{"rbrace", "}", []TokenType{TokenRBrace, TokenEOF}},
		{"caret", "^", []TokenType{TokenCaret, TokenEOF}},
		{"dollar", "$", []TokenType{TokenDollar, TokenEOF}},
		{"dash", "-", []TokenType{TokenDash, TokenEOF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(tokens) != len(tt.expected) {
				t.Errorf("Tokenize(%q) = %v, expected %v", tt.input, tokens, tt.expected)
				return
			}
			for i, expected := range tt.expected {
				if tokens[i].Type != expected {
					t.Errorf("Token[%d] = %v, want %v", i, tokens[i].Type, expected)
				}
			}
		})
	}
}

func TestTokenize_Escapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{"escaped dot", `\.`, []TokenType{TokenDot, TokenEOF}},
		{"escaped star", `\*`, []TokenType{TokenChar, TokenEOF}},
		{"escaped plus", `\+`, []TokenType{TokenPlus, TokenEOF}},
		{"escaped quest", `\?`, []TokenType{TokenQuest, TokenEOF}},
		{"escaped bar", `\|`, []TokenType{TokenBar, TokenEOF}},
		{"escaped parens", `\(\)`, []TokenType{TokenLParen, TokenRParen, TokenEOF}},
		{"escaped brackets", `\[\]`, []TokenType{TokenLBracket, TokenRBracket, TokenEOF}},
		{"escaped braces", `\{\}`, []TokenType{TokenLBrace, TokenRBrace, TokenEOF}},
		{"escaped caret", `\^`, []TokenType{TokenCaret, TokenEOF}},
		{"escaped dollar", `\$`, []TokenType{TokenDollar, TokenEOF}},
		{"escaped dash", `\-`, []TokenType{TokenDash, TokenEOF}},
		{"escaped backslash", `\\`, []TokenType{TokenChar, TokenEOF}},
		{"escape d", `\d`, []TokenType{TokenChar, TokenEOF}},
		{"escape D", `\D`, []TokenType{TokenChar, TokenEOF}},
		{"escape w", `\w`, []TokenType{TokenChar, TokenEOF}},
		{"escape W", `\W`, []TokenType{TokenChar, TokenEOF}},
		{"escape s", `\s`, []TokenType{TokenChar, TokenEOF}},
		{"escape S", `\S`, []TokenType{TokenChar, TokenEOF}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(tokens) != len(tt.expected) {
				t.Errorf("Tokenize(%q) = %v, expected %v", tt.input, tokens, tt.expected)
				return
			}
			for i, expected := range tt.expected {
				if tokens[i].Type != expected {
					t.Errorf("Token[%d] = %v, want %v", i, tokens[i].Type, expected)
				}
			}
		})
	}
}

func TestTokenize_CharacterClass(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTypes []TokenType
		wantClass []rune
	}{
		{
			name:      "simple class",
			input:     "[abc]",
			wantTypes: []TokenType{TokenLBracket, TokenEOF},
			wantClass: []rune{'a', 'b', 'c'},
		},
		{
			name:      "class with range",
			input:     "[a-z]",
			wantTypes: []TokenType{TokenLBracket, TokenEOF},
			wantClass: []rune{'a', '-', 'z'},
		},
		{
			name:      "negated class",
			input:     "[^abc]",
			wantTypes: []TokenType{TokenLBracket, TokenEOF},
			wantClass: []rune{'^', 'a', 'b', 'c'},
		},
		{
			name:      "negated class with range",
			input:     "[^a-z]",
			wantTypes: []TokenType{TokenLBracket, TokenEOF},
			wantClass: []rune{'^', 'a', '-', 'z'},
		},
		{
			name:      "mixed class",
			input:     "[a-zA-Z0-9]",
			wantTypes: []TokenType{TokenLBracket, TokenEOF},
			wantClass: []rune{'a', '-', 'z', 'A', '-', 'Z', '0', '-', '9'},
		},
		{
			name:      "class with escaped dash",
			input:     `[a\-z]`,
			wantTypes: []TokenType{TokenLBracket, TokenEOF},
			wantClass: []rune{'a', '-', 'z'}, // \- produces literal dash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(tokens) != len(tt.wantTypes) {
				t.Errorf("Tokenize(%q) = %v, expected %v types", tt.input, tokens, tt.wantTypes)
				return
			}
			if tokens[0].Type != tt.wantTypes[0] {
				t.Errorf("Token[0] = %v, want %v", tokens[0].Type, tt.wantTypes[0])
			}
			if len(tokens[0].Class) != len(tt.wantClass) {
				t.Errorf("Token[0].Class = %v (len %d), want %v (len %d)",
					tokens[0].Class, len(tokens[0].Class), tt.wantClass, len(tt.wantClass))
				return
			}
			for i, want := range tt.wantClass {
				if tokens[0].Class[i] != want {
					t.Errorf("Token[0].Class[%d] = %v, want %v", i, tokens[0].Class[i], want)
				}
			}
		})
	}
}

func TestTokenize_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"trailing backslash", `\`},
		{"unclosed bracket", `[abc`},
		{"unclosed bracket with escape", `[\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Tokenize(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
		})
	}
}

func TestLexer_NextPeek(t *testing.T) {
	l := New()
	l.Reset("abc")

	// First peek should return 'a'
	peeked := l.Peek()
	if peeked.Type != TokenChar || peeked.Val != 'a' {
		t.Errorf("Peek() = %v, want TokenChar 'a'", peeked)
	}

	// Next should return 'a'
	first := l.Next()
	if first.Type != TokenChar || first.Val != 'a' {
		t.Errorf("Next() = %v, want TokenChar 'a'", first)
	}

	// Peek again should return 'b' (not consuming)
	peeked = l.Peek()
	if peeked.Type != TokenChar || peeked.Val != 'b' {
		t.Errorf("Peek() = %v, want TokenChar 'b'", peeked)
	}

	// Next should return 'b'
	second := l.Next()
	if second.Type != TokenChar || second.Val != 'b' {
		t.Errorf("Next() = %v, want TokenChar 'b'", second)
	}

	// Peek should return 'c'
	peeked = l.Peek()
	if peeked.Type != TokenChar || peeked.Val != 'c' {
		t.Errorf("Peek() = %v, want TokenChar 'c'", peeked)
	}

	// Next should return 'c'
	third := l.Next()
	if third.Type != TokenChar || third.Val != 'c' {
		t.Errorf("Next() = %v, want TokenChar 'c'", third)
	}

	// Next should return EOF
	eof := l.Next()
	if eof.Type != TokenEOF {
		t.Errorf("Next() = %v, want TokenEOF", eof)
	}
}

func TestLexer_Reset(t *testing.T) {
	l := New()
	l.Reset("abc")

	if tok := l.Next(); tok.Type != TokenChar || tok.Val != 'a' {
		t.Errorf("First token = %v, want 'a'", tok)
	}

	// Reset and check from beginning
	l.Reset("xyz")
	if tok := l.Next(); tok.Type != TokenChar || tok.Val != 'x' {
		t.Errorf("After reset, first token = %v, want 'x'", tok)
	}
	if tok := l.Next(); tok.Type != TokenChar || tok.Val != 'y' {
		t.Errorf("After reset, second token = %v, want 'y'", tok)
	}
	if tok := l.Next(); tok.Type != TokenChar || tok.Val != 'z' {
		t.Errorf("After reset, third token = %v, want 'z'", tok)
	}
	if tok := l.Next(); tok.Type != TokenEOF {
		t.Errorf("After reset, fourth token = %v, want EOF", tok)
	}
}

func TestToken_IsQuantifier(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		expected  bool
	}{
		{TokenStar, true},
		{TokenPlus, true},
		{TokenQuest, true},
		{TokenChar, false},
		{TokenDot, false},
		{TokenBar, false},
	}

	for _, tt := range tests {
		tok := Token{Type: tt.tokenType}
		if tok.IsQuantifier() != tt.expected {
			t.Errorf("Token{%v}.IsQuantifier() = %v, want %v", tt.tokenType, tok.IsQuantifier(), tt.expected)
		}
	}
}

func TestTokenize_PositionAnchors(t *testing.T) {
	input := "^abc$"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := []TokenType{TokenCaret, TokenChar, TokenChar, TokenChar, TokenDollar, TokenEOF}
	if len(tokens) != len(expected) {
		t.Errorf("Tokenize(%q) = %v, expected %v", input, tokens, expected)
		return
	}

	for i, want := range expected {
		if tokens[i].Type != want {
			t.Errorf("Token[%d] = %v, want %v", i, tokens[i].Type, want)
		}
	}
}

func TestTokenize_CharacterClassWithNegation(t *testing.T) {
	// Test [^abc] - negated character class
	input := "[^abc]"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
		return
	}

	if tokens[0].Type != TokenLBracket {
		t.Errorf("Token[0] = %v, want TokenLBracket", tokens[0].Type)
	}

	// First element should be ^ (negation marker)
	if tokens[0].Class[0] != '^' {
		t.Errorf("Class[0] = %v, want '^'", tokens[0].Class[0])
	}
}

func TestTokenize_EscapeSequences(t *testing.T) {
	// Test all escape sequences
	input := `\d\D\w\W\s\S`
	tokens, err := Tokenize(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// All escapes should return TokenChar
	expectedTypes := []TokenType{TokenChar, TokenChar, TokenChar, TokenChar, TokenChar, TokenChar, TokenEOF}
	if len(tokens) != len(expectedTypes) {
		t.Errorf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
		return
	}

	for i, want := range expectedTypes {
		if tokens[i].Type != want {
			t.Errorf("Token[%d] = %v, want %v", i, tokens[i].Type, want)
		}
	}
}

func TestTokenize_EmptyClass(t *testing.T) {
	// Test [] - empty character class (valid in some regex flavors)
	input := "[]"
	tokens, err := Tokenize(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
		return
	}

	if tokens[0].Type != TokenLBracket {
		t.Errorf("Token[0] = %v, want TokenLBracket", tokens[0].Type)
	}

	// Empty class should have empty Class slice
	if len(tokens[0].Class) != 0 {
		t.Errorf("Token[0].Class = %v, want empty", tokens[0].Class)
	}
}

func TestTokenize_Repetition(t *testing.T) {
	// Test {n,m} repetition syntax - lexer produces individual char tokens inside braces
	tests := []struct {
		name  string
		input string
	}{
		{"exact", "a{3}"},
		{"range", "a{2,4}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Check first token is Char 'a'
			if tokens[0].Type != TokenChar || tokens[0].Val != 'a' {
				t.Errorf("First token = %v, want TokenChar 'a'", tokens[0])
			}
			// Check second token is LBrace
			if tokens[1].Type != TokenLBrace {
				t.Errorf("Second token = %v, want TokenLBrace", tokens[1].Type)
			}
			// Check last token is RBrace
			lastToken := tokens[len(tokens)-2] // -2 because last is EOF
			if lastToken.Type != TokenRBrace {
				t.Errorf("Last non-EOF token = %v, want TokenRBrace", lastToken.Type)
			}
		})
	}
}
