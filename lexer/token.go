package lexer

type TokenType int

const (
	TokenChar    TokenType = iota
	TokenDot
	TokenStar
	TokenPlus
	TokenQuest
	TokenBar
	TokenLParen
	TokenRParen
	TokenLBracket
	TokenRBracket
	TokenLBrace
	TokenRBrace
	TokenCaret
	TokenDollar
	TokenDash
	TokenEOF
	TokenError
	// Perl character classes
	TokenDigit  // \d
	TokenNDigit // \D
	TokenWord   // \w
	TokenNWord  // \W
	TokenSpace  // \s
	TokenNSpace // \S
)

type Token struct {
	Type  TokenType
	Pos   int
	Val   rune
	Class []rune
}

func (t *Token) IsQuantifier() bool {
	return t.Type == TokenStar || t.Type == TokenPlus || t.Type == TokenQuest
}
