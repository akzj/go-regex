package ast

type NodeType int

const (
	NodeChar       NodeType = iota
	NodeConcat
	NodeAlt
	NodeStar
	NodePlus
	NodeQuest
	NodeGroup
	NodeCapture
	NodeClass
	NodeNegClass
	NodeBegin
	NodeEnd
	NodeAny
	NodeRep
	NodeEmpty
)

type Node interface {
	Type() NodeType
	String() string
	Children() []Node
}

type CharNode struct {
	Ch rune
}

func (n *CharNode) Type() NodeType   { return NodeChar }
func (n *CharNode) String() string   { return string(n.Ch) }
func (n *CharNode) Children() []Node  { return nil }

type ClassNode struct {
	Ranges   []CharRange
	Negated  bool // true for [^...] patterns
}

func (n *ClassNode) Type() NodeType   { return NodeClass }
func (n *ClassNode) String() string   { return "[class]" }
func (n *ClassNode) Children() []Node { return nil }

type CharRange struct {
	Lo, Hi rune
}

type RepNode struct {
	Child Node
	Min   int
	Max   int
}

func (n *RepNode) Type() NodeType   { return NodeRep }
func (n *RepNode) String() string   { return "[rep]" }
func (n *RepNode) Children() []Node { return []Node{n.Child} }

// ConcatNode represents concatenation: A B
type ConcatNode struct {
	Left  Node
	Right Node
}

func (n *ConcatNode) Type() NodeType   { return NodeConcat }
func (n *ConcatNode) String() string   { return "[concat]" }
func (n *ConcatNode) Children() []Node { return []Node{n.Left, n.Right} }

// AltNode represents alternation: A | B
type AltNode struct {
	Left  Node
	Right Node
}

func (n *AltNode) Type() NodeType   { return NodeAlt }
func (n *AltNode) String() string   { return "[alt]" }
func (n *AltNode) Children() []Node { return []Node{n.Left, n.Right} }

// StarNode represents Kleene star: A*
type StarNode struct {
	Node Node
}

func (n *StarNode) Type() NodeType   { return NodeStar }
func (n *StarNode) String() string   { return "[star]" }
func (n *StarNode) Children() []Node { return []Node{n.Node} }

// PlusNode represents one or more: A+
type PlusNode struct {
	Node Node
}

func (n *PlusNode) Type() NodeType   { return NodePlus }
func (n *PlusNode) String() string   { return "[plus]" }
func (n *PlusNode) Children() []Node { return []Node{n.Node} }

// QuestNode represents optional: A?
type QuestNode struct {
	Node Node
}

func (n *QuestNode) Type() NodeType   { return NodeQuest }
func (n *QuestNode) String() string   { return "[quest]" }
func (n *QuestNode) Children() []Node { return []Node{n.Node} }

// GroupNode represents a non-capturing group: (?:A)
type GroupNode struct {
	Node Node
}

func (n *GroupNode) Type() NodeType   { return NodeGroup }
func (n *GroupNode) String() string   { return "[group]" }
func (n *GroupNode) Children() []Node { return []Node{n.Node} }

// CaptureNode represents a capturing group: (A)
type CaptureNode struct {
	Node  Node
	Index int
}

func (n *CaptureNode) Type() NodeType   { return NodeCapture }
func (n *CaptureNode) String() string   { return "[capture]" }
func (n *CaptureNode) Children() []Node { return []Node{n.Node} }

// BeginNode represents start of string/line anchor
type BeginNode struct{}

func (n *BeginNode) Type() NodeType   { return NodeBegin }
func (n *BeginNode) String() string { return "[begin]" }
func (n *BeginNode) Children() []Node { return nil }

// EndNode represents end of string/line anchor
type EndNode struct{}

func (n *EndNode) Type() NodeType   { return NodeEnd }
func (n *EndNode) String() string  { return "[end]" }
func (n *EndNode) Children() []Node { return nil }

// AnyNode represents wildcard match: .
type AnyNode struct{}

func (n *AnyNode) Type() NodeType   { return NodeAny }
func (n *AnyNode) String() string { return "[any]" }
func (n *AnyNode) Children() []Node { return nil }

// EmptyNode represents empty match
type EmptyNode struct{}

func (n *EmptyNode) Type() NodeType   { return NodeEmpty }
func (n *EmptyNode) String() string  { return "[empty]" }
func (n *EmptyNode) Children() []Node { return nil }
