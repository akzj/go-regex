package lexer

import "fmt"

// Lexer 词法分析器接口
type Lexer interface {
	Next() Token
	Peek() Token
	Reset(input string)
}

// lexer 词法分析器实现
type lexer struct {
	input  string
	pos    int
	start  int // for error reporting
}

// New 创建标准 lexer
func New() Lexer {
	return &lexer{}
}

// Reset 重置 lexer 状态
func (l *lexer) Reset(input string) {
	l.input = input
	l.pos = 0
	l.start = 0
}

// Next returns the next token
func (l *lexer) Next() Token {
	// Do NOT skip whitespace - it's a valid character in regex patterns
	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos}
	}
	return l.scanToken()
}

// Peek 返回下一个 token 但不消耗
func (l *lexer) Peek() Token {
	saved := l.pos
	tok := l.Next()
	l.pos = saved
	return tok
}

// scanToken 扫描单个 token
func (l *lexer) scanToken() Token {
	l.start = l.pos

	ch := rune(l.input[l.pos])

	// 转义序列
	if ch == '\\' {
		return l.scanEscape()
	}

	// 字符类开始
	if ch == '[' {
		return l.scanClass()
	}

	// 普通元字符
	switch ch {
	case '.':
		l.pos++
		return Token{Type: TokenDot, Pos: l.start, Val: '.'}
	case '*':
		l.pos++
		return Token{Type: TokenStar, Pos: l.start, Val: '*'}
	case '+':
		l.pos++
		return Token{Type: TokenPlus, Pos: l.start, Val: '+'}
	case '?':
		l.pos++
		return Token{Type: TokenQuest, Pos: l.start, Val: '?'}
	case '|':
		l.pos++
		return Token{Type: TokenBar, Pos: l.start, Val: '|'}
	case '(':
		l.pos++
		return Token{Type: TokenLParen, Pos: l.start, Val: '('}
	case ')':
		l.pos++
		return Token{Type: TokenRParen, Pos: l.start, Val: ')'}
	case '{':
		l.pos++
		return Token{Type: TokenLBrace, Pos: l.start, Val: '{'}
	case '}':
		l.pos++
		return Token{Type: TokenRBrace, Pos: l.start, Val: '}'}
	case '^':
		l.pos++
		return Token{Type: TokenCaret, Pos: l.start, Val: '^'}
	case '$':
		l.pos++
		return Token{Type: TokenDollar, Pos: l.start, Val: '$'}
	case ']':
		// Standalone ] is TokenRBracket
		l.pos++
		return Token{Type: TokenRBracket, Pos: l.start, Val: ']'}
	case '-':
		l.pos++
		return Token{Type: TokenDash, Pos: l.start, Val: '-'}
	}

	// 普通字符
	l.pos++
	return Token{Type: TokenChar, Pos: l.start, Val: ch}
}

// scanEscape 扫描转义序列
func (l *lexer) scanEscape() Token {
	l.pos++ // 消耗 '\'

	if l.pos >= len(l.input) {
		return Token{Type: TokenError, Pos: l.start, Val: '\\'}
	}

	ch := rune(l.input[l.pos])
	l.pos++ // 消耗转义字符

	switch ch {
	case 'd':
		return Token{Type: TokenDigit, Pos: l.start}
	case 'D':
		return Token{Type: TokenNDigit, Pos: l.start}
	case 'w':
		return Token{Type: TokenWord, Pos: l.start}
	case 'W':
		return Token{Type: TokenNWord, Pos: l.start}
	case 's':
		return Token{Type: TokenSpace, Pos: l.start}
	case 'S':
		return Token{Type: TokenNSpace, Pos: l.start}
	case '.':
		return Token{Type: TokenDot, Pos: l.start, Val: '.'}
	case '*':
		return Token{Type: TokenChar, Pos: l.start, Val: '*'}
	case '+':
		return Token{Type: TokenPlus, Pos: l.start, Val: '+'}
	case '?':
		return Token{Type: TokenQuest, Pos: l.start, Val: '?'}
	case '|':
		return Token{Type: TokenBar, Pos: l.start, Val: '|'}
	case '(':
		return Token{Type: TokenLParen, Pos: l.start, Val: '('}
	case ')':
		return Token{Type: TokenRParen, Pos: l.start, Val: ')'}
	case '[':
		return Token{Type: TokenLBracket, Pos: l.start, Val: '['}
	case ']':
		return Token{Type: TokenRBracket, Pos: l.start, Val: ']'}
	case '{':
		return Token{Type: TokenLBrace, Pos: l.start, Val: '{'}
	case '}':
		return Token{Type: TokenRBrace, Pos: l.start, Val: '}'}
	case '^':
		return Token{Type: TokenCaret, Pos: l.start, Val: '^'}
	case '$':
		return Token{Type: TokenDollar, Pos: l.start, Val: '$'}
	case '-':
		return Token{Type: TokenDash, Pos: l.start, Val: '-'}
	case '\\':
		return Token{Type: TokenChar, Pos: l.start, Val: '\\'}
	case 'n':
		return Token{Type: TokenChar, Pos: l.start, Val: '\n'}
	case 't':
		return Token{Type: TokenChar, Pos: l.start, Val: '\t'}
	case 'r':
		return Token{Type: TokenChar, Pos: l.start, Val: '\r'}
	default:
		// 未知转义序列，返回字符本身
		return Token{Type: TokenChar, Pos: l.start, Val: ch}
	}
}

// scanClass 扫描字符类 [....]
func (l *lexer) scanClass() Token {
	l.pos++ // 消耗 '['
	l.start = l.start // Keep original start position

	// 如果立即遇到 ]，这是空字符类 []
	if l.pos < len(l.input) && rune(l.input[l.pos]) == ']' {
		l.pos++ // 消耗 ']'
		return Token{Type: TokenLBracket, Pos: l.start, Class: []rune{}}
	}

	// 如果没有关闭的 ]，返回 TokenLBracket (无效的字符类)
	if l.pos >= len(l.input) {
		return Token{Type: TokenLBracket, Pos: l.start, Val: '['}
	}

	class := make([]rune, 0)

	// 检查是否是否定类 [^...]
	if rune(l.input[l.pos]) == '^' {
		class = append(class, '^')
		l.pos++
	}

	// 扫描字符类内容
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])

		if ch == ']' {
			// 字符类结束
			l.pos++ // 消耗 ']'
			return Token{Type: TokenLBracket, Pos: l.start, Class: class}
		}

		// 处理转义
		if ch == '\\' {
			l.pos++ // 消耗 '\'
			if l.pos >= len(l.input) {
				return Token{Type: TokenError, Pos: l.start, Val: '\\'}
			}
			esc := rune(l.input[l.pos])
			l.pos++
			class = append(class, esc)
			continue
		}

		// 检查是否是范围表达式 a-z
		// 范围只能在中间位置（有前一个字符），且后面不是 ]
		if len(class) > 0 && ch == '-' {
			nextPos := l.pos + 1
			if nextPos < len(l.input) && rune(l.input[nextPos]) != ']' {
				// 这是一个范围表达式: 前面已有 lo，现在取 hi
				l.pos++ // 消耗 '-'
				hi := rune(l.input[l.pos])
				l.pos++
				// 添加范围标记: '-' 表示范围开始，后跟上限
				class = append(class, '-')
				class = append(class, hi)
				continue
			}
		}

		// 普通字符
		l.pos++
		class = append(class, ch)
	}

	// 未闭合的字符类
	return Token{Type: TokenError, Pos: l.start, Val: '['}
}

// Tokenize 便捷函数，一次性返回所有 token
func Tokenize(input string) ([]Token, error) {
	l := New()
	l.Reset(input)

	var tokens []Token
	for {
		tok := l.Next()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}

	// 检查是否有错误
	for _, tok := range tokens {
		if tok.Type == TokenError {
			return tokens, fmt.Errorf("lexer error at position %d", tok.Pos)
		}
	}

	return tokens, nil
}
