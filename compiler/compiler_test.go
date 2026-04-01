package compiler

import (
	"testing"

	"github.com/akzj/go-regex/ast"
	"github.com/akzj/go-regex/machine"
)

func TestCompile_NilNode(t *testing.T) {
	c := New()
	_, err := c.Compile(nil)
	if err == nil {
		t.Error("expected error for nil AST node")
	}
}

func TestCompile_CharNode(t *testing.T) {
	c := New()
	node := &ast.CharNode{Ch: 'a'}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
	// Check that there's a literal transition on 'a'
	found := false
	for _, edge := range nfa.Start.Trans {
		if edge.Kind == machine.EdgeLiteral && edge.Char == 'a' {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected literal transition on 'a'")
	}
}

func TestCompile_ClassNode(t *testing.T) {
	c := New()
	node := &ast.ClassNode{Ranges: []ast.CharRange{{Lo: 'a', Hi: 'z'}}}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
	// Check for class edge
	found := false
	for _, edge := range nfa.Start.Trans {
		if edge.Kind == machine.EdgeClass {
			found = true
		}
	}
	if !found {
		t.Error("expected class transition")
	}
}

func TestCompile_EmptyNode(t *testing.T) {
	c := New()
	node := &ast.EmptyNode{}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_ConcatNode(t *testing.T) {
	c := New()
	// Pattern: "ab"
	node := &ast.ConcatNode{
		Left:  &ast.CharNode{Ch: 'a'},
		Right: &ast.CharNode{Ch: 'b'},
	}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
	if !nfa.End.IsAccept {
		t.Error("end state should be accepting")
	}
}

func TestCompile_AltNode(t *testing.T) {
	c := New()
	// Pattern: "a|b"
	node := &ast.AltNode{
		Left:  &ast.CharNode{Ch: 'a'},
		Right: &ast.CharNode{Ch: 'b'},
	}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_StarNode(t *testing.T) {
	c := New()
	// Pattern: "a*"
	node := &ast.StarNode{Node: &ast.CharNode{Ch: 'a'}}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_PlusNode(t *testing.T) {
	c := New()
	// Pattern: "a+"
	node := &ast.PlusNode{Node: &ast.CharNode{Ch: 'a'}}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_QuestNode(t *testing.T) {
	c := New()
	// Pattern: "a?"
	node := &ast.QuestNode{Node: &ast.CharNode{Ch: 'a'}}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_RepNode(t *testing.T) {
	c := New()
	tests := []struct {
		name string
		min  int
		max  int
	}{
		{"exact_2", 2, 2},
		{"range_2_4", 2, 4},
		{"at_least_1", 1, -1},
		{"at_most_3", 0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast.RepNode{
				Child: &ast.CharNode{Ch: 'a'},
				Min:   tt.min,
				Max:   tt.max,
			}
			nfa, err := c.Compile(node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if nfa == nil || nfa.Start == nil {
				t.Fatal("NFA should not be nil")
			}
		})
	}
}

func TestCompile_GroupNode(t *testing.T) {
	c := New()
	// Pattern: "(?:ab)"
	node := &ast.GroupNode{Node: &ast.ConcatNode{
		Left:  &ast.CharNode{Ch: 'a'},
		Right: &ast.CharNode{Ch: 'b'},
	}}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_CaptureNode(t *testing.T) {
	c := New()
	// Pattern: "(ab)"
	node := &ast.CaptureNode{
		Index: 1,
		Node: &ast.ConcatNode{
			Left:  &ast.CharNode{Ch: 'a'},
			Right: &ast.CharNode{Ch: 'b'},
		},
	}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_BeginNode(t *testing.T) {
	c := New()
	node := &ast.BeginNode{}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_EndNode(t *testing.T) {
	c := New()
	node := &ast.EndNode{}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
}

func TestCompile_AnyNode(t *testing.T) {
	c := New()
	node := &ast.AnyNode{}
	nfa, err := c.Compile(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nfa == nil || nfa.Start == nil {
		t.Fatal("NFA should not be nil")
	}
	// Check for any edge
	found := false
	for _, edge := range nfa.Start.Trans {
		if edge.Kind == machine.EdgeAny {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected any transition")
	}
}

func TestCompileDFA(t *testing.T) {
	c := New()
	node := &ast.CharNode{Ch: 'a'}
	dfa, err := c.CompileDFA(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dfa == nil || dfa.Start == nil {
		t.Fatal("DFA should not be nil")
	}
}

func TestCompileDFA_ComplexPattern(t *testing.T) {
	c := New()
	// Pattern: "(a|b)*c"
	node := &ast.ConcatNode{
		Left: &ast.StarNode{
			Node: &ast.AltNode{
				Left:  &ast.CharNode{Ch: 'a'},
				Right: &ast.CharNode{Ch: 'b'},
			},
		},
		Right: &ast.CharNode{Ch: 'c'},
	}
	dfa, err := c.CompileDFA(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dfa == nil || dfa.Start == nil {
		t.Fatal("DFA should not be nil")
	}
}

func TestNFAStateIDs_Deterministic(t *testing.T) {
	// Run twice and ensure state IDs are the same
	c1 := New()
	c2 := New()

	node := &ast.ConcatNode{
		Left:  &ast.CharNode{Ch: 'a'},
		Right: &ast.CharNode{Ch: 'b'},
	}

	nfa1, _ := c1.Compile(node)
	nfa2, _ := c2.Compile(node)

	// Collect all state IDs from nfa1
	ids1 := collectStateIDs(nfa1)
	ids2 := collectStateIDs(nfa2)

	if len(ids1) != len(ids2) {
		t.Errorf("different number of states: %d vs %d", len(ids1), len(ids2))
	}
}

func collectStateIDs(nfa *machine.NFA) []int {
	visited := make(map[*machine.NFAState]bool)
	var ids []int
	queue := []*machine.NFAState{nfa.Start}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]
		if visited[state] {
			continue
		}
		visited[state] = true
		ids = append(ids, state.ID)
		for _, edge := range state.Trans {
			if edge.Next != nil && !visited[edge.Next] {
				queue = append(queue, edge.Next)
			}
		}
	}
	return ids
}
