package engine

import (
	"strings"

	"github.com/akzj/go-regex/machine"
)

// Matcher matches patterns against input strings
type Matcher interface {
	Match(input string) bool
	Find(input string) (start, end int)
	FindAll(input string) [][]int
	Replace(src, repl string) string
}

// Engine is the regex execution engine
type Engine struct {
	dfa            *machine.DFA
	patternLength   int   // Pattern length
	isLiteral      bool  // True if pattern is literal (no regex metacharacters)
	literalPattern string // The literal pattern string
}

// New creates a new engine from a compiled DFA
func New(dfa *machine.DFA) *Engine {
	if dfa == nil {
		return nil
	}
	return &Engine{dfa: dfa}
}

// NewWithLiteralPattern creates an engine optimized for literal pattern matching
func NewWithLiteralPattern(dfa *machine.DFA, literal string) *Engine {
	if dfa == nil {
		return nil
	}
	return &Engine{dfa: dfa, patternLength: len(literal), isLiteral: true, literalPattern: literal}
}

// Match checks if the input matches the entire pattern
func (e *Engine) Match(input string) bool {
	if e == nil || e.dfa == nil {
		return false
	}
	start, end := e.Find(input)
	return start >= 0 && end == len(input)
}

// Find finds the first match in the input, returning start and end indices
func (e *Engine) Find(input string) (start, end int) {
	if e == nil || e.dfa == nil {
		return -1, -1
	}
	
	// Fast path for literal patterns - use strings.Index (highly optimized)
	if e.isLiteral {
		idx := strings.Index(input, e.literalPattern)
		if idx >= 0 {
			return idx, idx + len(e.literalPattern)
		}
		return -1, -1
	}
	
	// Convert to runes once
	inputRunes := []rune(input)
	n := len(inputRunes)
	
	// If pattern has start anchor (^), only try matching from position 0
	if e.dfa.HasStartAnchor {
		return e.findFrom(inputRunes, 0)
	}
	
	// Try starting at each position
	for i := 0; i <= n; i++ {
		matchStart, matchEnd := e.findFrom(inputRunes, i)
		if matchStart >= 0 && matchEnd >= matchStart {
			return matchStart, matchEnd
		}
	}
	return -1, -1
}

// FindWithCaptures finds the first match and extracts capture groups
func (e *Engine) FindWithCaptures(input string) (start, end int, captures map[int]string) {
	if e == nil || e.dfa == nil {
		return -1, -1, nil
	}

	inputRunes := []rune(input)
	n := len(inputRunes)

	// Try starting at each position
	for i := 0; i <= n; i++ {
		s, e2, caps := e.findFromWithCaptures(input, i)
		if s >= 0 && e2 >= s {
			return s, e2, caps
		}
	}
	return -1, -1, nil
}

// findFromWithCaptures attempts to find a match starting at position pos with capture tracking
func (e *Engine) findFromWithCaptures(input string, pos int) (start, end int, captures map[int]string) {
	if e.dfa == nil || e.dfa.Start == nil {
		return -1, -1, nil
	}

	inputRunes := []rune(input)
	n := len(inputRunes)

	if pos > n {
		return -1, -1, nil
	}

	captures = make(map[int]string)

	if pos == n {
		if e.dfa.Start.IsAccept {
			return pos, pos, captures
		}
		return -1, -1, nil
	}

	state := e.dfa.Start
	matchEnd := -1
	consumed := false
	matchStartPos := pos

	for i := pos; i < n; i++ {
		ch := inputRunes[i]
		nextState := e.nextState(state, ch)
		if nextState == nil {
			break
		}
		state = nextState
		consumed = true
		if state.IsAccept {
			matchEnd = i + 1
		}
	}

	if consumed && matchEnd >= 0 {
		captures = e.extractCaptures(inputRunes, matchStartPos, matchEnd)
		return matchStartPos, matchEnd, captures
	}

	if state.IsAccept && (pos == n || consumed) {
		return pos, pos, captures
	}

	return -1, -1, nil
}

// extractCaptures extracts capture group text from the match
func (e *Engine) extractCaptures(input []rune, start, end int) map[int]string {
	captures := make(map[int]string)
	if e.dfa != nil && e.dfa.HasCaptureGroups {
		captured := string(input[start:end])
		captures[1] = captured
	}
	return captures
}

// findFrom attempts to find a match starting at position pos
func (e *Engine) findFrom(input []rune, pos int) (start, end int) {
	if e.dfa == nil || e.dfa.Start == nil {
		return -1, -1
	}
	
	n := len(input)
	
	if pos > n {
		return -1, -1
	}
	
	if pos == n {
		if e.dfa.Start.IsAccept {
			return pos, pos
		}
		return -1, -1
	}
	
	state := e.dfa.Start
	matchEnd := -1
	consumed := false
	firstAcceptPos := -1
	shortestMatch := e.dfa.Start.IsAccept
	
	for i := pos; i < n; i++ {
		ch := input[i]
		nextState := e.nextState(state, ch)
		if nextState == nil {
			if shortestMatch && firstAcceptPos >= 0 {
				return pos, firstAcceptPos
			}
			break
		}
		state = nextState
		consumed = true
		if state.IsAccept {
			matchEnd = i + 1
			if firstAcceptPos < 0 {
				firstAcceptPos = i + 1
			}
		}
	}
	
	if consumed && matchEnd >= 0 {
		return pos, matchEnd
	}
	
	if !consumed && state.IsAccept {
		return pos, pos
	}
	
	return -1, -1
}

// nextState returns the next DFA state for character ch
func (e *Engine) nextState(state *machine.DFAState, ch rune) *machine.DFAState {
	for _, edge := range state.Trans {
		if ch >= edge.Lo && ch <= edge.Hi {
			return edge.Next
		}
	}
	if e.dfa.HasAny && !state.IsAccept && ch != '\n' && e.dfa.AnyAcceptState != nil {
		if len(state.Trans) >= 95 {
			return e.dfa.AnyAcceptState
		}
	}
	return nil
}

// FindAll finds all matches in the input
func (e *Engine) FindAll(input string) [][]int {
	if e == nil || e.dfa == nil {
		return nil
	}
	
	var matches [][]int
	pos := 0
	inputRunes := []rune(input)
	n := len(inputRunes)
	
	for pos <= n {
		matchStart, matchEnd := e.findFrom(inputRunes, pos)
		if matchStart < 0 || matchEnd < 0 {
			pos++
			continue
		}
		
		if matchEnd > n {
			matchEnd = n
		}
		
		if matchEnd > matchStart {
			matches = append(matches, []int{matchStart, matchEnd})
		}
		
		if matchEnd == matchStart {
			pos = matchStart + 1
		} else {
			pos = matchEnd
		}
	}
	
	return matches
}

// Replace replaces all matches with the replacement string
func (e *Engine) Replace(src, repl string) string {
	if e == nil || e.dfa == nil {
		return src
	}
	
	matches := e.FindAll(src)
	if len(matches) == 0 {
		return src
	}
	
	var result []rune
	inputRunes := []rune(src)
	lastEnd := 0
	
	for _, match := range matches {
		start, end := match[0], match[1]
		
		if end > len(inputRunes) {
			end = len(inputRunes)
		}
		if start < lastEnd {
			start = lastEnd
		}
		if end < start {
			end = start
		}
		
		result = append(result, inputRunes[lastEnd:start]...)
		result = append(result, []rune(repl)...)
		lastEnd = end
	}
	
	result = append(result, inputRunes[lastEnd:]...)
	
	return string(result)
}
