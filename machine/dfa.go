package machine

type AcceptInfo struct {
	GroupIndex  int
	CaptureStart int // Start position of this capture group in the match
	CaptureEnd   int // End position of this capture group in the match
}

type DFAState struct {
	ID          int
	NFASet      map[*NFAState]struct{}
	Trans       []DFAEdge
	IsAccept    bool
	AcceptInfos []AcceptInfo
}

type DFAEdge struct {
	Lo, Hi rune
	Next   *DFAState
}

type DFA struct {
	Start            *DFAState
	Alphabet         []rune
	HasStartAnchor   bool   // true if pattern begins with ^
	HasCaptureGroups  bool  // true if pattern has capture groups
	HasAny           bool   // true if pattern contains wildcard (.)
	AnyAcceptState   *DFAState // DFA state after consuming one char via wildcard (for Unicode fallback)
}
