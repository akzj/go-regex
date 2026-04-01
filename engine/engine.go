package engine

import (
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
	dfa *machine.DFA
}

// New creates a new engine from a compiled DFA
func New(dfa *machine.DFA) *Engine {
	if dfa == nil {
		return nil
	}
	return &Engine{dfa: dfa}
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
// Returns (-1, -1) if no match is found
func (e *Engine) Find(input string) (start, end int) {
	if e == nil || e.dfa == nil {
		return -1, -1
	}
	
	inputRunes := []rune(input)
	n := len(inputRunes)
	
	// If pattern has start anchor (^), only try matching from position 0
	if e.dfa.HasStartAnchor {
		return e.findFrom(input, 0)
	}
	
	// Try starting at each position
	for i := 0; i <= n; i++ {
		matchStart, matchEnd := e.findFrom(input, i)
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

	// If we're at end of string, check if current position accepts
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

	// Process each character starting from pos
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

	// Only return a match if we consumed input
	if consumed && matchEnd >= 0 {
		// Extract captures from match boundaries
		captures = e.extractCaptures(inputRunes, matchStartPos, matchEnd)
		return matchStartPos, matchEnd, captures
	}

	// Check for zero-length match
	if state.IsAccept && (pos == n || consumed) {
		return pos, pos, captures
	}

	return -1, -1, nil
}

// extractCaptures extracts capture group text from the match
func (e *Engine) extractCaptures(input []rune, start, end int) map[int]string {
	captures := make(map[int]string)
	
	// If the pattern has capture groups, derive them from the match
	if e.dfa != nil && e.dfa.HasCaptureGroups {
		// For simple patterns, capture group 1 = full match text
		captured := string(input[start:end])
		captures[1] = captured
	}
	
	return captures
}

// findFrom attempts to find a match starting at position pos
// Returns match indices if found, or (-1, -1) if no match
func (e *Engine) findFrom(input string, pos int) (start, end int) {
	if e.dfa == nil || e.dfa.Start == nil {
		return -1, -1
	}
	
	inputRunes := []rune(input)
	n := len(inputRunes)
	
	if pos > n {
		return -1, -1
	}
	
	// If we're at end of string, check if current position accepts
	if pos == n {
		if e.dfa.Start.IsAccept {
			return pos, pos // zero-length match at end
		}
		return -1, -1
	}
	
	state := e.dfa.Start
	matchEnd := -1
	consumed := false
	firstAcceptPos := -1 // Track first accepting position for shortest match
	
	// Determine matching strategy: patterns that accept empty (like a*) should
	// use shortest match to avoid greedily consuming too many chars.
	// Patterns that don't accept empty (like ab+) should use longest match.
	shortestMatch := e.dfa.Start.IsAccept
	
	// Process each character starting from pos
	for i := pos; i < n; i++ {
		ch := inputRunes[i]
		nextState := e.nextState(state, ch)
		if nextState == nil {
			// No transition for this character.
			// For shortest-match patterns (like a*), return at first accepting position.
			// For longest-match patterns (like ab+), use the last accepting position.
			if shortestMatch && firstAcceptPos >= 0 {
				return pos, firstAcceptPos
			}
			break
		}
		state = nextState
		consumed = true
		if state.IsAccept {
			matchEnd = i + 1
			// Record first accepting position
			if firstAcceptPos < 0 {
				firstAcceptPos = i + 1
			}
		}
	}
	
	// Only return a match if we consumed input
	if consumed && matchEnd >= 0 {
		return pos, matchEnd
	}
	
	// Check for zero-length match only at start if pattern can match empty
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
	// Use wildcard fallback only when:
	// 1. Pattern has wildcard (.)
	// 2. Character is not newline (which . never matches)
	// 3. Current state is NOT already accepting (don't consume past end of match)
	// 4. State has WIDE transitions (many chars, indicating wildcard pattern)
	//    If state has few transitions, it's an explicit character pattern - don't use fallback
	if e.dfa.HasAny && !state.IsAccept && ch != '\n' && e.dfa.AnyAcceptState != nil {
		// Only apply fallback if this state has wide transitions (95+ = ASCII printable range)
		// This distinguishes wildcard patterns (.) from explicit char patterns (a.b)
		if len(state.Trans) >= 95 {
			return e.dfa.AnyAcceptState
		}
	}
	return nil
}

// FindAll finds all matches in the input, returning rune indices
func (e *Engine) FindAll(input string) [][]int {
	if e == nil || e.dfa == nil {
		return nil
	}
	
	var matches [][]int
	pos := 0
	inputRunes := []rune(input)
	n := len(inputRunes)
	
	for pos <= n {
		matchStart, matchEnd := e.findFrom(input, pos)
		if matchStart < 0 || matchEnd < 0 {
			// No match at this position - try next position
			pos++
			continue
		}
		
		// Clamp bounds
		if matchEnd > n {
			matchEnd = n
		}
		
		// Only record if we matched something (non-zero length)
		if matchEnd > matchStart {
			matches = append(matches, []int{matchStart, matchEnd})
		}
		
		// Advance position - for zero-length matches, advance by 1
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
		
		// Clamp bounds
		if end > len(inputRunes) {
			end = len(inputRunes)
		}
		if start < lastEnd {
			start = lastEnd
		}
		if end < start {
			end = start
		}
		
		// Append text before match
		result = append(result, inputRunes[lastEnd:start]...)
		// Append replacement
		result = append(result, []rune(repl)...)
		lastEnd = end
	}
	
	// Append remaining text
	result = append(result, inputRunes[lastEnd:]...)
	
	return string(result)
}
