package machine

import "sync"

// EdgeKind represents the type of NFA edge
type EdgeKind int

const (
	EdgeEpsilon EdgeKind = iota
	EdgeLiteral
	EdgeAny
	EdgeClass
	EdgeNegClass  // Negated character class: [^...]
	EdgeCaptureStart // Mark the start of a capture group
	EdgeCaptureEnd   // Mark the end of a capture group
)

// NFAState represents a state in the NFA
// Invariant: Epsilon transitions are always prioritized over character transitions
type NFAState struct {
	ID       int
	Label    string
	Trans    []NFAEdge
	IsAccept bool
}

// NFAEdge represents a transition in the NFA
type NFAEdge struct {
	Kind      EdgeKind
	Char      rune
	Next      *NFAState
	Min       int
	Max       int
	GroupIndex int // Capture group index for EdgeCaptureStart/End
}

// Fragment represents an NFA fragment (used for Thompson construction)
// Invariant: Start and End are never nil
type Fragment struct {
	Start *NFAState
	End   *NFAState
}

// NFA is a complete Non-deterministic Finite Automaton
type NFA struct {
	Start *NFAState
	End   *NFAState
}

// NFA construction state - shared counter for state IDs
var (
	stateID   int
	stateLock sync.Mutex
)

// newState creates a new NFA state with a unique ID
// The state is created with an optional label for debugging
func newState(label string) *NFAState {
	stateLock.Lock()
	defer stateLock.Unlock()
	stateID++
	return &NFAState{
		ID:    stateID,
		Label: label,
		Trans: make([]NFAEdge, 0),
	}
}

// addEdge adds an edge from from to to state
func addEdge(from, to *NFAState, edge NFAEdge) {
	edge.Next = to
	from.Trans = append(from.Trans, edge)
}

// epsilon creates an epsilon transition fragment: ε
// Returns a fragment with start →ε→ end
func epsilon() Fragment {
	start := newState("epsilon")
	end := newState("epsilon_end")
	addEdge(start, end, NFAEdge{Kind: EdgeEpsilon})
	return Fragment{Start: start, End: end}
}

// literal creates a fragment matching a single character: 'c'
// Returns a fragment with start →c→ end
func literal(c rune) Fragment {
	start := newState("literal")
	end := newState("literal_end")
	addEdge(start, end, NFAEdge{Kind: EdgeLiteral, Char: c})
	return Fragment{Start: start, End: end}
}

// any creates a fragment matching any single character: .
// Returns a fragment with start →any→ end
func any() Fragment {
	start := newState("any")
	end := newState("any_end")
	addEdge(start, end, NFAEdge{Kind: EdgeAny})
	return Fragment{Start: start, End: end}
}

// class creates a fragment matching a character from a class
// min/max define the character range [min, max]
func class(min, max rune) Fragment {
	start := newState("class")
	end := newState("class_end")
	addEdge(start, end, NFAEdge{Kind: EdgeClass, Char: min, Min: int(min), Max: int(max)})
	return Fragment{Start: start, End: end}
}

// Connect connects two fragments in sequence: A·B
// Connects A.End to B.Start with an epsilon transition
func Connect(a, b Fragment) Fragment {
	addEdge(a.End, b.Start, NFAEdge{Kind: EdgeEpsilon})
	return Fragment{Start: a.Start, End: b.End}
}

// Or creates an alternation: A|B
// Creates new start and end states with epsilon transitions to both alternatives
func Or(a, b Fragment) Fragment {
	start := newState("or_start")
	end := newState("or_end")

	// start →ε→ a.Start and start →ε→ b.Start
	addEdge(start, a.Start, NFAEdge{Kind: EdgeEpsilon})
	addEdge(start, b.Start, NFAEdge{Kind: EdgeEpsilon})

	// a.End →ε→ end and b.End →ε→ end
	addEdge(a.End, end, NFAEdge{Kind: EdgeEpsilon})
	addEdge(b.End, end, NFAEdge{Kind: EdgeEpsilon})

	return Fragment{Start: start, End: end}
}

// Star creates Kleene star: A*
// Creates loops allowing zero or more repetitions
func Star(a Fragment) Fragment {
	start := newState("star_start")
	end := newState("star_end")

	// Loop: a.End →ε→ a.Start (allow repeating)
	addEdge(a.End, a.Start, NFAEdge{Kind: EdgeEpsilon})

	// Skip: start →ε→ end (allow zero occurrences)
	addEdge(start, end, NFAEdge{Kind: EdgeEpsilon})

	// Enter: start →ε→ a.Start
	addEdge(start, a.Start, NFAEdge{Kind: EdgeEpsilon})

	// Exit: a.End →ε→ end
	addEdge(a.End, end, NFAEdge{Kind: EdgeEpsilon})

	return Fragment{Start: start, End: end}
}

// Plus creates positive closure: A+
// Requires at least one occurrence, then allows zero or more repetitions
func Plus(a Fragment) Fragment {
	// For A+, a.End becomes the accepting state
	// We add a loop from a.End back to a.Start to allow repetitions
	// The structure is: start →ε→ a.Start →...→ a.End ←ε← (loop back)
	// After first match of A, we're at a.End which is accepting
	
	// Loop: a.End →ε→ a.Start (allow repeating)
	addEdge(a.End, a.Start, NFAEdge{Kind: EdgeEpsilon})
	
	// Enter from new start state
	start := newState("plus_start")
	addEdge(start, a.Start, NFAEdge{Kind: EdgeEpsilon})
	
	// The accepting state is a.End (no separate end state needed)
	// This prevents epsilon-only paths to acceptance
	return Fragment{Start: start, End: a.End}
}

// Quest creates optional: A?
// Allows zero or one occurrence
func Quest(a Fragment) Fragment {
	start := newState("quest_start")
	end := newState("quest_end")

	// Skip: start →ε→ end (zero occurrences)
	addEdge(start, end, NFAEdge{Kind: EdgeEpsilon})

	// Match: start →ε→ a.Start →ε→ end
	addEdge(start, a.Start, NFAEdge{Kind: EdgeEpsilon})
	addEdge(a.End, end, NFAEdge{Kind: EdgeEpsilon})

	return Fragment{Start: start, End: end}
}

// Rep creates repetition with bounds: A{min,max}
// Rep(min=-1, max=-1) is equivalent to Star
// Rep(min=1, max=1) is equivalent to Plus without loop
// Rep(min=0, max=1) is equivalent to Quest
// Rep(min=0, max=-1) is equivalent to Star
// Rep(min=n, max=m) where n>0 allows n to m repetitions
func Rep(a Fragment, min, max int) Fragment {
	// Special cases for common patterns
	if min == 0 && max == 1 {
		return Quest(a)
	}
	if min == 0 && max == -1 {
		return Star(a)
	}
	if min == 1 && max == -1 {
		return Plus(a)
	}
	if min == 1 && max == 1 {
		return a
	}

	// General case: build from epsilon fragments
	start := newState("rep_start")
	end := newState("rep_end")

	// Build optional prefix for min occurrences
	prefix := Fragment{Start: start, End: start}
	for i := 0; i < min; i++ {
		prefix = Connect(prefix, a)
	}

	// Build optional suffix for additional max-min occurrences
	if max == -1 {
		// Unlimited: A{min,∞} = A{min} A*
		suffix := Star(a)
		result := Connect(prefix, suffix)
		addEdge(result.End, end, NFAEdge{Kind: EdgeEpsilon})
		return Fragment{Start: start, End: end}
	}

	// Limited: A{min,max} = A{min} (A{min,max-min})?
	additional := Fragment{Start: start, End: start}
	for i := 0; i < max-min; i++ {
		additional = Connect(additional, Quest(a))
	}

	result := Connect(prefix, additional)
	addEdge(result.End, end, NFAEdge{Kind: EdgeEpsilon})

	return Fragment{Start: start, End: end}
}

// ResetStateID resets the global state ID counter for deterministic testing
func ResetStateID() {
	stateLock.Lock()
	defer stateLock.Unlock()
	stateID = 0
}

// Exported helper functions for Thompson construction
// These wrap the lowercase versions for use by the compiler package

// Epsilon creates an epsilon transition fragment: ε
func Epsilon() Fragment {
	return epsilon()
}

// Literal creates a fragment matching a single character: 'c'
func Literal(c rune) Fragment {
	return literal(c)
}

// Any creates a fragment matching any single character: .
func Any() Fragment {
	return any()
}

// Class creates a fragment matching a character from a class
// min/max define the character range [min, max]
func Class(min, max rune) Fragment {
	return class(min, max)
}

// NegClass creates a fragment matching a character NOT in a class
// min/max define the excluded character range [min, max]
func NegClass(min, max rune) Fragment {
	start := newState("negclass")
	end := newState("negclass_end")
	addEdge(start, end, NFAEdge{Kind: EdgeNegClass, Char: min, Min: int(min), Max: int(max)})
	return Fragment{Start: start, End: end}
}

// CaptureStart creates a fragment that marks the start of a capture group
// Returns epsilon fragment with capture start marker
func CaptureStart(groupIndex int) Fragment {
	start := newState("capture_start")
	end := newState("capture_start_end")
	addEdge(start, end, NFAEdge{Kind: EdgeCaptureStart, GroupIndex: groupIndex})
	return Fragment{Start: start, End: end}
}

// CaptureEnd creates a fragment that marks the end of a capture group
// Returns epsilon fragment with capture end marker
func CaptureEnd(groupIndex int) Fragment {
	start := newState("capture_end")
	end := newState("capture_end_end")
	addEdge(start, end, NFAEdge{Kind: EdgeCaptureEnd, GroupIndex: groupIndex})
	return Fragment{Start: start, End: end}
}
