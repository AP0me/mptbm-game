package shared

import (
	"fmt"
	"strings"
)

// ANSI constants used by the tree renderer
const (
	Dim          = "\033[2m"
	BrightCyan   = "\033[96m"
	BrightYellow = "\033[93m"
	BrightGreen  = "\033[92m"
	BrightWhite  = "\033[97m"
	BrightBlue   = "\033[94m"
	BrightRed    = "\033[91m"
)

// wrap applies ANSI codes then resets.
func wrap(text string, codes ...string) string {
	return strings.Join(codes, "") + text + Reset
}

// DirectedTree mirrors the Python DirectedTree.
// Branches maps a branch name to an ordered list of milestone strings.
// Selections maps a branch name to the current cursor index (-1 = nothing done).
// BranchOrder preserves the intended display order of branches.
type DirectedTree struct {
	Branches    map[string][]string `json:"branches"`
	Selections  map[string]int      `json:"selections"`
	BranchOrder []string            `json:"branch_order"`
}

func NewDirectedTree(branches map[string][]string, selections map[string]int, order []string) *DirectedTree {
	return &DirectedTree{
		Branches:    branches,
		Selections:  selections,
		BranchOrder: order,
	}
}

// indexOf returns the index of childKey within the branch, or -1 if not found.
func (t *DirectedTree) indexOf(layer1Key, childKey string) int {
	kids, ok := t.Branches[layer1Key]
	if !ok {
		return -1
	}
	for i, v := range kids {
		if v == childKey {
			return i
		}
	}
	return -1
}

// IsAdvanced returns true if the branch cursor has reached or passed childKey.
func (t *DirectedTree) IsAdvanced(layer1Key, childKey string) bool {
	current := t.Selections[layer1Key]
	target := t.indexOf(layer1Key, childKey)
	return current >= target
}

// AdvanceTo moves the branch cursor forward to childKey (never backward).
func (t *DirectedTree) AdvanceTo(layer1Key, childKey string) {
	if !t.IsAdvanced(layer1Key, childKey) {
		t.Selections[layer1Key] = t.indexOf(layer1Key, childKey)
	}
}

// CurrentChild returns the milestone at the current cursor for the branch.
func (t *DirectedTree) CurrentChild(layer1Key string) string {
	idx := t.Selections[layer1Key]
	kids := t.Branches[layer1Key]
	if idx < 0 || idx >= len(kids) {
		return ""
	}
	return kids[idx]
}

// Render produces the full ANSI tree visualization, matching the Python output.
func (t *DirectedTree) Render(title string) string {
	var lines []string
	lines = append(lines, "")
	bar := strings.Repeat("=", 60)

	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, wrap("  "+title, BrightCyan, Bold))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	// Root node
	lines = append(lines, wrap("  root", BrightGreen, Bold))
	lines = append(lines, "   │")

	for bIdx, layer1Key := range t.BranchOrder {
		kids := t.Branches[layer1Key]
		currentIdx := t.Selections[layer1Key]
		isLastBranch := bIdx == len(t.BranchOrder)-1

		connector := "├──"
		if isLastBranch {
			connector = "└──"
		}
		lines = append(lines,
			"  "+wrap(connector+" ", Cyan)+wrap(layer1Key, BrightYellow, Bold))

		vpipe := "│  "
		if isLastBranch {
			vpipe = "   "
		}
		vpipeStyled := wrap(vpipe, Cyan)

		for i, kid := range kids {
			isLastKid := i == len(kids)-1
			kidConnector := "├──"
			if isLastKid {
				kidConnector = "└──"
			}
			connectorStyled := wrap(kidConnector, Cyan)

			var marker, kidName string
			if i <= currentIdx {
				marker = wrap("[✓]", Dim, Green)
				kidName = wrap(kid, Green)
			} else {
				marker = wrap("[ ]", Dim, Gray)
				kidName = wrap(kid, Dim)
			}

			lines = append(lines,
				"  "+vpipeStyled+connectorStyled+" "+marker+" "+kidName)
		}

		if !isLastBranch {
			lines = append(lines, wrap("  │", Cyan))
		}
	}

	lines = append(lines, "")
	// Legend
	lines = append(lines,
		wrap("  Legend: ", Dim)+
			wrap("[✓]", Dim, Green)+
			wrap(" done   ", Dim)+
			wrap("[ ]", Dim, Gray)+
			wrap(" not done", Dim))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// Dump is a plain-text debug dump (no ANSI).
func (t *DirectedTree) Dump() string {
	var b strings.Builder
	for _, k := range t.BranchOrder {
		idx := t.Selections[k]
		for i, c := range t.Branches[k] {
			if i == idx {
				fmt.Fprintf(&b, "  %s -> %s *\n", k, c)
			} else if i < idx {
				fmt.Fprintf(&b, "  %s -> %s (learned)\n", k, c)
			} else {
				fmt.Fprintf(&b, "  %s -> %s\n", k, c)
			}
		}
	}
	return b.String()
}
