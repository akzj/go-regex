package api

import (
	"github.com/akzj/go-regex/compiler"
	"github.com/akzj/go-regex/engine"
	"github.com/akzj/go-regex/parser"
)

// Regex represents a compiled regular expression
type Regex struct {
	engine *engine.Engine
	expr   string
}

// Compile compiles a regular expression pattern
// Returns a *Regex that can be used for matching
// Error conditions:
//   - Invalid regex syntax
//   - Unsupported regex features
//   - Empty pattern (returns EmptyRegex)
func Compile(pattern string) (*Regex, error) {
	if pattern == "" {
		// Return a regex that never matches
		return &Regex{engine: nil, expr: ""}, nil
	}

	// Parse the pattern
	p := parser.New()
	node, err := p.ParseString(pattern)
	if err != nil {
		return nil, err
	}

	// Compile to DFA
	c := compiler.New()
	dfa, err := c.CompileDFA(node)
	if err != nil {
		return nil, err
	}

	// Create engine
	e := engine.New(dfa)

	return &Regex{
		engine: e,
		expr:   pattern,
	}, nil
}

// MustCompile compiles a regular expression and panics on error
func MustCompile(pattern string) *Regex {
	r, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return r
}

// Match reports whether the Regex matches the entire input string s
func (r *Regex) Match(s string) bool {
	if r.engine == nil {
		// Empty pattern never matches (except empty string)
		return s == ""
	}
	return r.engine.Match(s)
}

// Find returns a string holding the text of the leftmost match in s
// Returns empty string if no match
func (r *Regex) Find(s string) string {
	if r.engine == nil {
		return ""
	}
	start, end := r.engine.Find(s)
	if start < 0 {
		return ""
	}
	// Convert to runes since engine.Find returns rune indices
	inputRunes := []rune(s)
	return string(inputRunes[start:end])
}

// FindStringSubmatch returns a slice of strings holding the text of
// the leftmost match in s and the matches, if any, within that match.
func (r *Regex) FindStringSubmatch(s string) []string {
	if r.engine == nil {
		return nil
	}
	start, end, captures := r.engine.FindWithCaptures(s)
	if start < 0 {
		return nil
	}
	
	inputRunes := []rune(s)
	fullMatch := string(inputRunes[start:end])
	
	// Build result: [fullMatch, group1, group2, ...]
	result := []string{fullMatch}
	
	// Add capture groups in order (group 1, 2, 3, ...)
	for i := 1; ; i++ {
		if capText, ok := captures[i]; ok {
			result = append(result, capText)
		} else {
			break
		}
	}
	
	return result
}

// FindAll returns a slice of all non-overlapping matches.
// Each match is a slice of strings holding the text of the leftmost
// match and the matches, if any, within that match.
func (r *Regex) FindAll(s string) [][]int {
	if r.engine == nil {
		return nil
	}
	return r.engine.FindAll(s)
}

// ReplaceAllString returns a copy of s, replacing all non-overlapping
// matches of the Regex with the replacement string repl
func (r *Regex) ReplaceAllString(s, repl string) string {
	if r.engine == nil {
		return s
	}
	return r.engine.Replace(s, repl)
}

// Split slices s into substrings separated by the Regex
// and returns a slice of the substrings between those expression matches.
func (r *Regex) Split(s string) []string {
	if s == "" {
		return []string{}
	}
	
	if r.engine == nil {
		// No pattern: return the original string in a slice
		return []string{s}
	}

	matches := r.engine.FindAll(s)
	if len(matches) == 0 {
		return []string{s}
	}

	var result []string
	lastEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]
		result = append(result, s[lastEnd:start])
		lastEnd = end
	}

	result = append(result, s[lastEnd:])
	return result
}

// String returns the source text of the compiled regular expression
func (r *Regex) String() string {
	return r.expr
}
