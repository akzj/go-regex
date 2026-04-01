package compiler

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/akzj/go-regex/ast"
	"github.com/akzj/go-regex/machine"
)

// Compiler compiles AST to NFA/DFA
type Compiler interface {
	Compile(node ast.Node) (*machine.NFA, error)
	CompileDFA(node ast.Node) (*machine.DFA, error)
}

// compiler implements the Compiler interface using Thompson construction
type compiler struct {
	stateID      int
	stateLock    sync.Mutex
	captureIndex int // tracks the next capture group index
}

// New creates a new Compiler instance
func New() Compiler {
	return &compiler{}
}

// Compile compiles an AST into an NFA using Thompson construction
func (c *compiler) Compile(node ast.Node) (*machine.NFA, error) {
	if node == nil {
		return nil, errors.New("nil AST node")
	}
	c.resetStateID()
	frag := c.compileNode(node)
	if frag.Start == nil {
		return nil, errors.New("failed to compile AST node")
	}
	frag.End.IsAccept = true
	return &machine.NFA{Start: frag.Start, End: frag.End}, nil
}

// CompileDFA compiles an AST into a DFA
func (c *compiler) CompileDFA(node ast.Node) (*machine.DFA, error) {
	nfa, err := c.Compile(node)
	if err != nil {
		return nil, err
	}
	return c.nfaToDFA(nfa)
}

func (c *compiler) resetStateID() {
	machine.ResetStateID()
	c.captureIndex = 0
}

func (c *compiler) compileNode(node ast.Node) machine.Fragment {
	if node == nil {
		return machine.Fragment{}
	}

	switch n := node.(type) {
	case *ast.CharNode:
		return c.compileChar(n)
	case *ast.ClassNode:
		return c.compileClass(n)
	case *ast.RepNode:
		return c.compileRep(n)
	case *ast.ConcatNode:
		return c.compileConcat(n)
	case *ast.AltNode:
		return c.compileAlt(n)
	case *ast.StarNode:
		return c.compileStar(n)
	case *ast.PlusNode:
		return c.compilePlus(n)
	case *ast.QuestNode:
		return c.compileQuest(n)
	case *ast.GroupNode:
		return c.compileGroup(n)
	case *ast.CaptureNode:
		return c.compileCapture(n)
	case *ast.BeginNode:
		return c.compileBegin(n)
	case *ast.EndNode:
		return c.compileEnd(n)
	case *ast.AnyNode:
		return c.compileAny(n)
	case *ast.EmptyNode:
		return c.compileEmpty(n)
	default:
		panic(fmt.Sprintf("unknown AST node type: %T", node))
	}
}

func (c *compiler) compileChar(n *ast.CharNode) machine.Fragment {
	return machine.Literal(n.Ch)
}

func (c *compiler) compileClass(n *ast.ClassNode) machine.Fragment {
	if len(n.Ranges) == 0 {
		return machine.Epsilon()
	}

	// Handle negated classes
	if n.Negated {
		if len(n.Ranges) == 1 {
			return machine.NegClass(n.Ranges[0].Lo, n.Ranges[0].Hi)
		}
		// For multiple ranges, use first range's span
		return machine.NegClass(n.Ranges[0].Lo, n.Ranges[len(n.Ranges)-1].Hi)
	}

	if len(n.Ranges) == 1 {
		return machine.Class(n.Ranges[0].Lo, n.Ranges[0].Hi)
	}

	var result machine.Fragment
	for i, r := range n.Ranges {
		frag := machine.Class(r.Lo, r.Hi)
		if i == 0 {
			result = frag
		} else {
			result = machine.Or(result, frag)
		}
	}
	return result
}

func (c *compiler) compileRep(n *ast.RepNode) machine.Fragment {
	child := c.compileNode(n.Child)
	return machine.Rep(child, n.Min, n.Max)
}

func (c *compiler) compileConcat(n *ast.ConcatNode) machine.Fragment {
	left := c.compileNode(n.Left)
	right := c.compileNode(n.Right)
	return machine.Connect(left, right)
}

func (c *compiler) compileAlt(n *ast.AltNode) machine.Fragment {
	left := c.compileNode(n.Left)
	right := c.compileNode(n.Right)
	return machine.Or(left, right)
}

func (c *compiler) compileStar(n *ast.StarNode) machine.Fragment {
	child := c.compileNode(n.Node)
	return machine.Star(child)
}

func (c *compiler) compilePlus(n *ast.PlusNode) machine.Fragment {
	child := c.compileNode(n.Node)
	return machine.Plus(child)
}

func (c *compiler) compileQuest(n *ast.QuestNode) machine.Fragment {
	child := c.compileNode(n.Node)
	return machine.Quest(child)
}

func (c *compiler) compileGroup(n *ast.GroupNode) machine.Fragment {
	return c.compileNode(n.Node)
}

func (c *compiler) compileCapture(n *ast.CaptureNode) machine.Fragment {
	// Get the group index from the AST node (set by parser)
	groupIndex := n.Index
	if groupIndex == 0 {
		groupIndex = c.captureIndex
		c.captureIndex++
	}

	// Compile the child fragment normally
	child := c.compileNode(n.Node)

	// Mark that this pattern has capture groups
	// (We'll handle capture extraction differently)
	
	return child
}

func (c *compiler) compileBegin(n *ast.BeginNode) machine.Fragment {
	return machine.Epsilon()
}

func (c *compiler) compileEnd(n *ast.EndNode) machine.Fragment {
	return machine.Epsilon()
}

func (c *compiler) compileAny(n *ast.AnyNode) machine.Fragment {
	return machine.Any()
}

func (c *compiler) compileEmpty(n *ast.EmptyNode) machine.Fragment {
	return machine.Epsilon()
}

// epsilonClosure computes the epsilon closure of a set of NFA states
func (c *compiler) epsilonClosure(states map[*machine.NFAState]struct{}) map[*machine.NFAState]struct{} {
	closure := make(map[*machine.NFAState]struct{})
	stack := make([]*machine.NFAState, 0)

	for s := range states {
		closure[s] = struct{}{}
		stack = append(stack, s)
	}

	for len(stack) > 0 {
		state := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, edge := range state.Trans {
			if edge.Kind == machine.EdgeEpsilon {
				next := edge.Next
				if _, ok := closure[next]; !ok {
					closure[next] = struct{}{}
					stack = append(stack, next)
				}
			}
		}
	}

	return closure
}

// move computes the transition on input character
func (c *compiler) move(states map[*machine.NFAState]struct{}, ch rune) map[*machine.NFAState]struct{} {
	result := make(map[*machine.NFAState]struct{})

	for state := range states {
		for _, edge := range state.Trans {
			switch edge.Kind {
			case machine.EdgeLiteral:
				if edge.Char == ch {
					result[edge.Next] = struct{}{}
				}
			case machine.EdgeAny:
				if ch != '\n' {
					result[edge.Next] = struct{}{}
				}
			case machine.EdgeClass:
				if int(ch) >= edge.Min && int(ch) <= edge.Max {
					result[edge.Next] = struct{}{}
				}
			case machine.EdgeNegClass:
				// NegClass matches any character NOT in the range
				if int(ch) < edge.Min || int(ch) > edge.Max {
					result[edge.Next] = struct{}{}
				}
			}
		}
	}

	return result
}

// nfaToDFA converts an NFA to DFA using subset construction
func (c *compiler) nfaToDFA(nfa *machine.NFA) (*machine.DFA, error) {
	initialSet := make(map[*machine.NFAState]struct{})
	initialSet[nfa.Start] = struct{}{}
	initialClosure := c.epsilonClosure(initialSet)

	dfaStateID := 0

	startState := &machine.DFAState{
		ID:        dfaStateID,
		NFASet:    initialClosure,
		IsAccept:  c.containsAcceptState(initialClosure),
	}
	dfaStateID++

	dfaStates := []*machine.DFAState{startState}
	unmarked := []*machine.DFAState{startState}

	alphabet, hasAny := c.buildAlphabet(nfa)

	// Check if pattern starts with ^ (begin anchor)
	hasStartAnchor := c.startsWithBegin(nfa.Start)

	for len(unmarked) > 0 {
		current := unmarked[len(unmarked)-1]
		unmarked = unmarked[:len(unmarked)-1]

		for _, ch := range alphabet {
			moveResult := c.move(current.NFASet, ch)
			if len(moveResult) == 0 {
				continue
			}

			closure := c.epsilonClosure(moveResult)

			found := false
			for _, state := range dfaStates {
				if c.nfaSetEqual(state.NFASet, closure) {
					current.Trans = append(current.Trans, machine.DFAEdge{
						Lo:   ch,
						Hi:   ch,
						Next: state,
					})
					found = true
					break
				}
			}

			if !found {
				newState := &machine.DFAState{
					ID:       dfaStateID,
					NFASet:   closure,
					IsAccept: c.containsAcceptState(closure),
				}
				dfaStateID++

				current.Trans = append(current.Trans, machine.DFAEdge{
					Lo:   ch,
					Hi:   ch,
					Next: newState,
				})

				dfaStates = append(dfaStates, newState)
				unmarked = append(unmarked, newState)
			}
		}
	}

	// Find the AnyAcceptState - the accepting state reached after consuming
	// one character via wildcard (. pattern). This is used for Unicode fallback.
	var anyAcceptState *machine.DFAState
	if hasAny {
		for _, state := range dfaStates {
			if state.IsAccept {
				anyAcceptState = state
				break
			}
		}
	}

	return &machine.DFA{
		Start:             startState,
		Alphabet:          alphabet,
		HasStartAnchor:    hasStartAnchor,
		HasCaptureGroups:  c.captureIndex > 0,
		HasAny:            hasAny,
		AnyAcceptState:    anyAcceptState,
	}, nil
}

func (c *compiler) containsAcceptState(states map[*machine.NFAState]struct{}) bool {
	for state := range states {
		if state.IsAccept {
			return true
		}
	}
	return false
}

// extractAcceptInfos extracts capture group information from NFA states
// prevSet is the NFA state set before consuming input, currSet is after
func (c *compiler) extractAcceptInfos(prevSet, currSet map[*machine.NFAState]struct{}) []machine.AcceptInfo {
	var infos []machine.AcceptInfo
	
	// Look for EdgeCaptureEnd edges in the current state set
	// These indicate the end of a capture group
	for state := range currSet {
		for _, edge := range state.Trans {
			if edge.Kind == machine.EdgeCaptureEnd {
				infos = append(infos, machine.AcceptInfo{
					GroupIndex: edge.GroupIndex,
				})
			}
		}
	}
	
	return infos
}

// startsWithBegin checks if the NFA starts with a Begin (^) anchor.
// It detects this by examining if the start state has ONLY epsilon transitions
// (BeginNode is compiled as epsilon), and those epsilons eventually lead to
// character-consuming states. This distinguishes ^ from other constructs like *.
//
// However, for alternation (a|b), the or_start state also has only epsilon
// transitions. We distinguish alternation by checking if there are multiple
// entry points to character-consuming states in the epsilon closure.
func (c *compiler) startsWithBegin(start *machine.NFAState) bool {
	if start == nil {
		return false
	}

	hasNonEpsilon := false

	for _, edge := range start.Trans {
		if edge.Kind != machine.EdgeEpsilon {
			hasNonEpsilon = true
			break
		}
	}

	// If start has non-epsilon transitions, it's definitely not a pure start anchor
	if hasNonEpsilon {
		return false
	}

	// Start has only epsilon transitions.
	// Now check if there are multiple entry points to character-consuming states.
	// If so, it's alternation (a|b), not a start anchor.
	closure := c.epsilonClosure(map[*machine.NFAState]struct{}{start: {}})

	// Count how many distinct character-consuming states are reachable
	charEntryStates := make(map[*machine.NFAState]struct{})
	for state := range closure {
		for _, edge := range state.Trans {
			if edge.Kind != machine.EdgeEpsilon {
				// This state has a character-consuming transition
				charEntryStates[state] = struct{}{}
			}
		}
	}

	// If there are multiple entry points, it's alternation, not a start anchor
	return len(charEntryStates) == 1
}

func (c *compiler) nfaSetEqual(a, b map[*machine.NFAState]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// buildAlphabet extracts all unique characters from NFA transitions
// Returns the alphabet and whether the pattern contains a wildcard (.)
func (c *compiler) buildAlphabet(nfa *machine.NFA) ([]rune, bool) {
	charSet := make(map[rune]struct{})
	hasAny := false
	hasNegClass := false

	visited := make(map[*machine.NFAState]bool)
	queue := []*machine.NFAState{nfa.Start}
	visited[nfa.Start] = true

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		for _, edge := range state.Trans {
			switch edge.Kind {
			case machine.EdgeLiteral:
				charSet[edge.Char] = struct{}{}
			case machine.EdgeAny:
				hasAny = true
			case machine.EdgeClass:
				for ch := rune(edge.Min); ch <= rune(edge.Max); ch++ {
					charSet[ch] = struct{}{}
				}
			case machine.EdgeNegClass:
				hasNegClass = true
				// Add the excluded range to alphabet (even though we negate it)
				for ch := rune(edge.Min); ch <= rune(edge.Max); ch++ {
					charSet[ch] = struct{}{}
				}
			}

			// Follow all transitions for alphabet building (not just epsilon)
			if !visited[edge.Next] {
				visited[edge.Next] = true
				queue = append(queue, edge.Next)
			}
		}
	}

	// If we have a wildcard (. pattern), add ASCII characters to the alphabet
	// so DFA transitions are built for common inputs.
	// The HasAny flag will handle Unicode characters at runtime.
	if hasAny {
		for ch := rune(32); ch <= 126; ch++ {
			charSet[ch] = struct{}{}
		}
	}

	// If we have negated character classes, add representative characters
	// from the complement to ensure DFA has transitions for those inputs.
	// This is needed because negated class [^a-z] should match '0', 'A', etc.
	if hasNegClass {
		// Add printable ASCII range (32-126) to ensure DFA has transitions
		// for common characters outside typical ranges
		for ch := rune(32); ch <= 126; ch++ {
			charSet[ch] = struct{}{}
		}
	}

	alphabet := make([]rune, 0, len(charSet))
	for ch := range charSet {
		alphabet = append(alphabet, ch)
	}
	sort.Slice(alphabet, func(i, j int) bool { return alphabet[i] < alphabet[j] })

	return alphabet, hasAny
}
